// Тест шва фронта (см. docs/SPEC.md, шов фронтенда):
// SSE-клиент + плеер — взаимодействие сети и UI.
// Проверяем, что клиент правильно парсит поток и кладёт кадры в плеер,
// и что на HTTP-ошибку приходит понятное сообщение, а не кадры с мусором.
import { describe, expect, it, vi } from "vitest";
import { createPlayer } from "../src/player";
import { createStream } from "../src/stream";

/** Собрать ответ с заранее известными байтами SSE-потока. */
function sseResponse(events: string[]): Response {
  const body = events.join("\n\n") + "\n\n";
  const bytes = new TextEncoder().encode(body);
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      controller.enqueue(bytes);
      controller.close();
    },
  });
  return new Response(stream, {
    status: 200,
    headers: { "Content-Type": "text/event-stream; charset=utf-8" },
  });
}

const sampleFrame = (step: number): string =>
  `data: {"step":${step},"kind":"array","explanation":"шаг ${step}","data":{"values":[1,3,5],"low":0,"high":2,"mid":-1,"foundIndex":-1}}`;

describe("createStream", () => {
  it("читает поток и кладёт все кадры в плеер", async () => {
    const player = createPlayer([]);
    const onFrame = vi.fn();
    const stream = createStream({
      url: "/api/run/binary-search?values=1,3,5&target=3",
      player,
      sink: { onFrame },
      fetchImpl: (() =>
        Promise.resolve(sseResponse([sampleFrame(1), sampleFrame(2), sampleFrame(3)]))) as typeof fetch,
    });
    await stream.start();

    expect(player.currentIndex()).toBe(0);
    expect(player.currentFrame()?.step).toBe(1);
    expect(onFrame).toHaveBeenCalled();
    expect(onFrame).toHaveBeenLastCalledWith(3);
  });

  it("на HTTP 400 показывает ошибку из JSON-тела, не роняет плеер", async () => {
    const player = createPlayer([]);
    const onError = vi.fn();
    const stream = createStream({
      url: "/api/run/binary-search?values=5,1,3&target=3",
      player,
      sink: { onError },
      fetchImpl: (() =>
        Promise.resolve(
          new Response(JSON.stringify({ error: "массив должен быть отсортирован" }), {
            status: 400,
            headers: { "Content-Type": "application/json" },
          }),
        )) as typeof fetch,
    });

    await stream.start();

    expect(onError).toHaveBeenCalledWith("массив должен быть отсортирован");
    expect(player.currentFrame()).toBeNull();
  });

  it("на сетевую ошибку вызывает onError и не валит плеер", async () => {
    const player = createPlayer([]);
    const onError = vi.fn();
    const stream = createStream({
      url: "/api/run/binary-search?values=1&target=1",
      player,
      sink: { onError },
      fetchImpl: (() => Promise.reject(new Error("connection refused"))) as typeof fetch,
    });

    await stream.start();

    expect(onError).toHaveBeenCalledWith(expect.stringContaining("connection refused"));
    expect(player.currentFrame()).toBeNull();
  });
});