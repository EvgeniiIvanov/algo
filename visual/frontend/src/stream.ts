// SSE-клиент: тянет поток кадров с бэкенда и складывает в плеер.
//
// Контракт:
//   createStream(url, player, sink) → start/stop.
//
// Поток читается как ReadableStream<Uint8Array> из fetch с Accept: text/event-stream.
// На каждое событие `data: {...}\n\n` парсим кадр и кладём в плеер через player.load.
// На любую ошибку — уведомляем sink.error().
//
// Шов между сетью и UI — этот модуль и плеер. Тесты — в tests/stream.test.ts.

import { parseError, parseFrame } from "./sse";
import type { Player } from "./player";

export interface StreamSink {
  onFrame?: (count: number) => void;
  onError?: (message: string) => void;
  onEnd?: () => void;
}

/** Опции для создания SSE-клиента; выделены, чтобы тестам было легко подсунуть fetch. */
export interface StreamOptions {
  url: string;
  player: Player;
  sink?: StreamSink;
  /** Внедряется для тестов. В проде — глобальный fetch. */
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
      options.player.load([]);

      let resp: Response;
      try {
        resp = await fetchImpl(options.url, {
          headers: { Accept: "text/event-stream" },
          signal: controller.signal,
        });
      } catch (err) {
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
      const collected: import("./types").Frame[] = [];

      try {
        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          // Сервер присылает кадры разделённые \n\n — режем по ним,
          // остаток в buffer до следующего read.
          let boundary = buffer.indexOf("\n\n");
          while (boundary !== -1) {
            const event = buffer.slice(0, boundary);
            buffer = buffer.slice(boundary + 2);
            if (event.length > 0) {
              try {
                const frame = parseFrame(event);
                collected.push(frame);
              } catch (err) {
                options.sink?.onError?.(`кадр отброшен: ${(err as Error).message}`);
              }
            }
            boundary = buffer.indexOf("\n\n");
          }
          // Накопленные кадры — в плеер. На каждом пакете reader,
          // а не на последнем: так пользователь видит прогресс стрима.
          if (collected.length > 0) {
            options.player.load([...collected]);
            options.sink?.onFrame?.(collected.length);
          }
        }
      } catch (err) {
        if ((err as { name?: string }).name === "AbortError") {
          // Это остановка через stop() — не ошибка.
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
          collected.push(parseFrame(tail));
          options.player.load([...collected]);
        } catch (err) {
          options.sink?.onError?.(`хвост отброшен: ${(err as Error).message}`);
        }
      }
      options.sink?.onEnd?.();
    },
    stop(): void {
      controller?.abort();
      controller = null;
    },
  };
}