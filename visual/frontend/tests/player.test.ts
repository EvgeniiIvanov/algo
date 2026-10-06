// Тест шва фронта (см. docs/SPEC.md, шов фронтенда):
// плеер — управление списком кадров: play/pause/seek/stepForward/stepBackward/setSpeed.
// Это шов: тестируем только публичные методы и подписки, не DOM и не таймеры.
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createPlayer } from "../src/player";
import type { ArraySnapshot } from "../src/types";

const snapshot = (over: Partial<ArraySnapshot> = {}): ArraySnapshot => ({
  values: [1, 3, 5, 7, 9],
  low: 0,
  high: 4,
  mid: -1,
  foundIndex: -1,
  ...over,
});

const frames = [
  { step: 1, kind: "array", explanation: "старт", data: snapshot() },
  { step: 2, kind: "array", explanation: "середина", data: snapshot({ mid: 2 }) },
  { step: 3, kind: "array", explanation: "правая", data: snapshot({ low: 3 }) },
  { step: 4, kind: "array", explanation: "найдено", data: snapshot({ foundIndex: 3 }) },
];

describe("createPlayer", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("стартует в первом кадре и на паузе", () => {
    const p = createPlayer(frames);
    expect(p.isPlaying()).toBe(false);
    expect(p.currentFrame()).toEqual(frames[0]);
    expect(p.currentIndex()).toBe(0);
    expect(p.framesCount()).toBe(frames.length);
  });

  it("stepForward продвигает кадр вперёд; на последнем — остаётся", () => {
    const p = createPlayer(frames);
    p.stepForward();
    expect(p.currentIndex()).toBe(1);
    p.stepForward();
    p.stepForward();
    p.stepForward(); // за последним — не двигается
    expect(p.currentIndex()).toBe(3);
  });

  it("stepBackward отступает назад; на нулевом — остаётся", () => {
    const p = createPlayer(frames);
    p.seek(2);
    p.stepBackward();
    expect(p.currentIndex()).toBe(1);
    p.stepBackward();
    p.stepBackward();
    p.stepBackward(); // до нуля
    expect(p.currentIndex()).toBe(0);
    p.stepBackward(); // за ноль
    expect(p.currentIndex()).toBe(0);
  });

  it("seek(индекс) переходит к нужному кадру и зажимает буфер", () => {
    const p = createPlayer(frames);
    p.seek(2);
    expect(p.currentIndex()).toBe(2);
    expect(p.currentFrame()).toEqual(frames[2]);
    p.seek(99);
    expect(p.currentIndex()).toBe(3);
    p.seek(-1);
    expect(p.currentIndex()).toBe(0);
  });

  it("setSpeed задаёт скорость, getSpeed отдаёт её", () => {
    const p = createPlayer(frames);
    expect(p.getSpeed()).toBeGreaterThanOrEqual(1);
    p.setSpeed(4);
    expect(p.getSpeed()).toBe(4);
    p.setSpeed(0); // нельзя — минимум 1
    expect(p.getSpeed()).toBe(1);
    p.setSpeed(99); // зажимаем сверху
    expect(p.getSpeed()).toBeLessThanOrEqual(16);
  });

  it("play/pause: при play кадры идут по таймеру; pause останавливает", () => {
    const p = createPlayer(frames);
    p.setSpeed(2);
    p.play();
    expect(p.isPlaying()).toBe(true);

    vi.advanceTimersByTime(1500); // 1.5 секунды реального времени → 3 кадра при 2х
    expect(p.currentIndex()).toBeGreaterThanOrEqual(2);

    p.pause();
    const idxBefore = p.currentIndex();
    vi.advanceTimersByTime(2000);
    expect(p.currentIndex()).toBe(idxBefore);
  });

  it("play сам останавливается на последнем кадре", () => {
    const p = createPlayer(frames);
    p.seek(3);
    p.play();
    expect(p.isPlaying()).toBe(false); // уже конец
  });

  it("notify подписчиков при смене кадра", () => {
    const p = createPlayer(frames);
    const listener = vi.fn();
    p.subscribe(listener);
    p.stepForward();
    p.seek(2);
    expect(listener).toHaveBeenCalledTimes(2);
    expect(listener).toHaveBeenLastCalledWith(frames[2]);
  });

  it("load заменяет кадры и сбрасывает позицию в 0", () => {
    const p = createPlayer(frames);
    p.seek(2);
    const fresh = [
      { step: 1, kind: "array", explanation: "новый старт", data: snapshot() },
      { step: 2, kind: "array", explanation: "новый шаг", data: snapshot() },
    ];
    p.load(fresh);
    expect(p.currentIndex()).toBe(0);
    expect(p.currentFrame()).toEqual(fresh[0]);
  });

  it("пустой список кадров не падает; currentFrame === null", () => {
    const p = createPlayer([]);
    expect(p.currentFrame()).toBeNull();
    expect(p.currentIndex()).toBe(0);
    p.play();
    expect(p.isPlaying()).toBe(false);
    p.stepForward();
    expect(p.currentIndex()).toBe(0);
  });

  // Регрессия на P2 #2: при авто-паузе на последнем кадре UI должен узнать о смене состояния.
  it("play до конца — на последнем кадре останавливается, notify идёт уже с playing=false", () => {
    const p = createPlayer(frames);
    const states: boolean[] = [];
    p.subscribe(() => {
      states.push(p.isPlaying());
    });
    p.setSpeed(2);
    p.play();

    // Прокрутим достаточно времени, чтобы пройти 4 кадра при 2x (по 500мс на шаг).
    vi.advanceTimersByTime(10_000);

    expect(p.currentIndex()).toBe(frames.length - 1);
    expect(p.isPlaying()).toBe(false);
    // Все нотификации на шагах 1..4 были во время play; финальная — на паузе.
    expect(states[states.length - 1]).toBe(false);
  });

  // Регрессия на P2 #1: новые кадры через append не сбрасывают позицию и не останавливают play.
  it("append добавляет кадры в хвост, не сбрасывает позицию и не прерывает play", () => {
    const p = createPlayer(frames.slice(0, 2)); // кадры 1..2, индекс 0
    p.seek(1);
    expect(p.currentIndex()).toBe(1);

    p.append([frames[2], frames[3]]);
    expect(p.framesCount()).toBe(4);
    expect(p.currentIndex()).toBe(1); // позиция сохранилась
    expect(p.isPlaying()).toBe(false); // append сам по себе не запускает play

    p.play();
    expect(p.isPlaying()).toBe(true);

    // Пришли ещё кадры во время проигрывания — play не должен прерываться,
    // позиция не должна сбрасываться.
    const more = {
      ...frames[0],
      step: 5,
      explanation: "ещё один",
    } as const;
    p.append([more]);
    expect(p.framesCount()).toBe(5);
    expect(p.isPlaying()).toBe(true);
  });

  // Регрессия на P1 (ревью-2): append обязан уведомить подписчика, даже если плеер не играет.
  // Иначе после нового Run кадры приходят, но UI их не видит и панель не показывается.
  it("append уведомляет подписчика, даже если плеер не играет", () => {
    const p = createPlayer([]);
    const listener = vi.fn();
    p.subscribe(listener);

    p.append([frames[0]]);
    expect(listener).toHaveBeenCalledTimes(1);
    expect(listener).toHaveBeenLastCalledWith(frames[0]);

    p.append([frames[1], frames[2]]);
    // Один notify на каждый append, не на каждый кадр. Что рендерить — решает
    // рендерер: текущий кадр не сменился, рендерер отдаст currentFrame().
    expect(listener).toHaveBeenCalledTimes(2);
  });

  // Регрессия на P1 (ревью-2): append на непустом плеере сохраняет позицию.
  it("append на непустом плеере сохраняет позицию", () => {
    const p = createPlayer(frames.slice(0, 2)); // кадры 1..2
    expect(p.currentIndex()).toBe(0);
    p.seek(1);
    expect(p.currentIndex()).toBe(1);

    p.append([frames[2], frames[3]]);
    expect(p.framesCount()).toBe(4);
    expect(p.currentIndex()).toBe(1);
    expect(p.currentFrame()).toEqual(frames[1]);
  });
});