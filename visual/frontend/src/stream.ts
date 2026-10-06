// SSE-клиент: тянет поток кадров с бэкенда и складывает в плеер.
//
// Контракт:
//   createStream(url, player, sink) → start/stop.
//
// Поток читается как ReadableStream<Uint8Array> из fetch с Accept: text/event-stream.
// Кадры накапливаются и:
//   • первый кадр — через player.load (стартовая позиция);
//   • последующие — через player.append (позиция пользователя сохраняется).
// На ошибку 4xx/5xx — уведомляем sink.onError с сообщением из JSON-тела.
// На AbortError (наш stop()) — тихо завершаемся, без ошибки.
//
// Шов между сетью и UI — этот модуль и плеер. Тесты — в tests/stream.test.ts.

import { parseError, parseFrame } from "./sse";
import type { Player } from "./player";
import type { Frame } from "./types";

export interface StreamSink {
  onFrame?: (count: number) => void;
  onError?: (message: string) => void;
  onEnd?: () => void;
}

export interface StreamOptions {
  url: string;
  player: Player;
  sink?: StreamSink;
  fetchImpl?: typeof fetch;
}

export interface Stream {
  start(): Promise<void>;
  stop(): void;
}

export function createStream(options: StreamOptions): Stream {
  const fetchImpl = options.fetchImpl ?? fetch;
  let controller: AbortController | null = null;

  return {
    async start(): Promise<void> {
      controller = new AbortController();
      // Не вызываем player.load([]) здесь — мы хотим добавлять кадры в хвост.
      // Если в плеере что-то лежало от предыдущего Run, ответственность на main.ts
      // (он создаёт новый плеер на новый Run). Здесь только стрим.

      let resp: Response;
      try {
        resp = await fetchImpl(options.url, {
          headers: { Accept: "text/event-stream" },
          signal: controller.signal,
        });
      } catch (err) {
        // Наш собственный AbortController: тихий выход, без onError.
        if ((err as { name?: string }).name === "AbortError") return;
        options.sink?.onError?.(`не удалось подключиться: ${(err as Error).message}`);
        return;
      }

      if (!resp.ok) {
        let detail = "";
        try {
          detail = await resp.text();
        } catch {
          // пустое тело — оставляем как есть
        }
        const friendly = parseError(detail) ?? `сервер вернул ${resp.status}`;
        options.sink?.onError?.(friendly);
        return;
      }
      if (resp.body === null) {
        options.sink?.onError?.("сервер не прислал тело потока");
        return;
      }

      const reader = resp.body.getReader();
      const decoder = new TextDecoder("utf-8");
      let buffer = "";
      // Кадры, разобранные в текущей жизни потока. В плеер уходит только «хвост»,
      // которого там нет — это страховка от дублирования, если бэкенд пришлёт
      // кадры порциями. Хранить весь список приходится потому, что SSE
      // не гарантирует доставку пакетов по границам кадров — один кадр может
      // прийти в соседних чанках.
      const allFrames: Frame[] = [];

      try {
        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          // Сервер присылает кадры, разделённые \n\n — режем по ним,
          // остаток в buffer до следующего read.
          let boundary = buffer.indexOf("\n\n");
          while (boundary !== -1) {
            const event = buffer.slice(0, boundary);
            buffer = buffer.slice(boundary + 2);
            if (event.length > 0) {
              try {
                allFrames.push(parseFrame(event));
              } catch (err) {
                options.sink?.onError?.(`кадр отброшен: ${(err as Error).message}`);
              }
            }
            boundary = buffer.indexOf("\n\n");
          }
          // Отдаём плееру только новые кадры — пользователь видит прогресс
          // по мере прихода, но каждый кадр попадает в плеер ровно один раз.
          if (allFrames.length > options.player.framesCount()) {
            const delta = allFrames.slice(options.player.framesCount());
            options.player.append(delta);
            options.sink?.onFrame?.(options.player.framesCount());
          }
        }
      } catch (err) {
        if ((err as { name?: string }).name === "AbortError") {
          options.sink?.onEnd?.();
          return;
        }
        options.sink?.onError?.(`стрим прерван: ${(err as Error).message}`);
        return;
      }

      // Хвост без \n\n — пробуем распарсить, если что-то есть.
      const tail = buffer.trim();
      if (tail.length > 0) {
        try {
          allFrames.push(parseFrame(tail));
        } catch (err) {
          options.sink?.onError?.(`хвост отброшен: ${(err as Error).message}`);
        }
      }

      // Финальный прогон: если в последнем чанке пришли кадры, которых ещё нет в плеере.
      if (allFrames.length > options.player.framesCount()) {
        const delta = allFrames.slice(options.player.framesCount());
        options.player.append(delta);
        options.sink?.onFrame?.(options.player.framesCount());
      }
      options.sink?.onEnd?.();
    },
    stop(): void {
      controller?.abort();
      controller = null;
    },
  };
}