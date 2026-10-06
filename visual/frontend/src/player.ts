// Плеер кадров (см. docs/SPEC.md, шов фронтенда):
// хранит список уже полученных кадров и управляет позицией.
// Не знает про fetch и DOM — только данные и таймер.
//
// Контракт:
//   createPlayer(frames) → объект с методами play/pause/stepForward/stepBackward/
//   seek/setSpeed/getSpeed/currentFrame/currentIndex/isPlaying/subscribe/
//   load/append/framesCount.
//
// Два способа получить кадры:
//   load(frames)  — заменить весь список и сбросить позицию к нулю (новый запуск Run).
//   append(frames) — добавить в хвост, не трогая позицию и play (кадры пришли по SSE).
//
// Шов не опускается ниже: notify делается на каждую смену кадра, не «внутри».

import type { Frame } from "./types";

const MIN_SPEED = 1;
const MAX_SPEED = 16;
/** Базовая длительность одного шага при скорости 1x. */
const BASE_STEP_MS = 1000;

export interface Player {
  play(): void;
  pause(): void;
  isPlaying(): boolean;
  stepForward(): void;
  stepBackward(): void;
  /** seek(index): перейти к кадру по индексу; индекс зажимается в [0, length-1]. */
  seek(index: number): void;
  setSpeed(speed: number): void;
  getSpeed(): number;
  currentIndex(): number;
  currentFrame(): Frame | null;
  /** Сколько кадров сейчас в плеере. */
  framesCount(): number;
  /** Заменить все кадры и сбросить позицию/таймер. Использовать при новом Run. */
  load(frames: Frame[]): void;
  /** Добавить кадры в хвост, не трогая позицию и play. Использовать при поступлении кадров из SSE. */
  append(frames: Frame[]): void;
  subscribe(listener: (frame: Frame) => void): () => void;
}

export function createPlayer(initial: Frame[]): Player {
  let frames: Frame[] = [...initial];
  let index = 0;
  let speed = 1;
  let playing = false;
  let timer: ReturnType<typeof setTimeout> | null = null;
  const listeners = new Set<(frame: Frame) => void>();

  const currentFrame = (): Frame | null => (frames.length === 0 ? null : frames[index] ?? null);

  const notify = (): void => {
    const frame = currentFrame();
    if (frame === null) return;
    for (const listener of listeners) listener(frame);
  };

  const stepIntervalMs = (): number => Math.max(1, Math.round(BASE_STEP_MS / speed));

  const stopTimer = (): void => {
    if (timer !== null) {
      clearTimeout(timer);
      timer = null;
    }
  };

  /** Остановить воспроизведение: и таймер, и флаг playing. */
  const pause = (): void => {
    playing = false;
    stopTimer();
  };

  const scheduleNext = (): void => {
    stopTimer();
    if (!playing || index >= frames.length - 1) return;
    timer = setTimeout(() => {
      timer = null;
      if (!playing) return;
      // Дошли до последнего кадра — пауза, и только потом notify,
      // чтобы подписчик (UI) увидел playing=false в том же notify.
      if (index >= frames.length - 1) {
        playing = false;
        notify();
        return;
      }
      index++;
      notify();
      if (playing && index < frames.length - 1) scheduleNext();
      else {
        playing = false;
        notify(); // финальный кадр уже отправлен выше; здесь сигнал UI о паузе
      }
    }, stepIntervalMs());
  };

  return {
    play(): void {
      if (playing || frames.length === 0 || index >= frames.length - 1) return;
      playing = true;
      scheduleNext();
    },
    pause,
    isPlaying(): boolean {
      return playing;
    },
    stepForward(): void {
      if (frames.length === 0) return;
      pause();
      if (index < frames.length - 1) {
        index++;
        notify();
      }
    },
    stepBackward(): void {
      if (frames.length === 0) return;
      pause();
      if (index > 0) {
        index--;
        notify();
      }
    },
    seek(target: number): void {
      if (frames.length === 0) return;
      pause();
      const clamped = Math.max(0, Math.min(frames.length - 1, Math.trunc(target)));
      if (clamped !== index) {
        index = clamped;
        notify();
      }
    },
    setSpeed(value: number): void {
      const clamped = Math.max(MIN_SPEED, Math.min(MAX_SPEED, Math.trunc(value)));
      if (clamped === speed) return;
      const wasPlaying = playing;
      stopTimer();
      speed = clamped;
      if (wasPlaying) scheduleNext();
    },
    getSpeed(): number {
      return speed;
    },
    currentIndex(): number {
      return index;
    },
    framesCount(): number {
      return frames.length;
    },
    currentFrame,
    load(next: Frame[]): void {
      pause();
      frames = [...next];
      index = 0;
      notify();
    },
    append(next: Frame[]): void {
      // Хвост без сброса позиции и без прерывания play — кадры пришли из SSE.
      if (next.length === 0) return;
      frames = [...frames, ...next];
      // Если плеер стоит в конце списка и не играет — пользователь ещё не нажал play;
      // не дёргаем notify: кадров за пределами курсора он не увидит, а лишний апдейт
      // может сбить UI (например, seek«сбрасывается»). Если играет — scheduleNext
      // сам подхватит новые кадры на следующей итерации.
      if (playing) scheduleNext();
    },
    subscribe(listener: (frame: Frame) => void): () => void {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
  };
}