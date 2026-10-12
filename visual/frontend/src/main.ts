// Точка входа: связывает форму ввода, плеер и SSE-клиент.
// Слои:
//   форма → createStream(url, player, sink)
//   плеер → renderFrame(frame, explanation, array) на каждый notify.

import { createPlayer } from "./player";
import { createStream } from "./stream";
import { renderFrame } from "./render";
import { buildQueueRunURL, buildRunURL, buildStackRunURL } from "./sse";
import { parseValues, parseTarget } from "./binarysearch";
import { parseQueueCapacity, parseQueueScenario, parseStackCapacity, parseStackScenario } from "./scenario";
import type { Frame } from "./types";

const ALGORITHMS = ["binary-search", "stack", "queue"] as const;
type Algorithm = (typeof ALGORITHMS)[number];

function getElement<T extends HTMLElement>(id: string): T {
  const el = document.getElementById(id);
  if (el === null) throw new Error(`id=${id} не найден в DOM`);
  return el as T;
}

function init(): void {
  const explanationEl = getElement<HTMLElement>("explanation");
  const arrayEl = getElement<HTMLElement>("array");
  const algorithmEl = getElement<HTMLSelectElement>("algorithm");
  const valuesEl = getElement<HTMLInputElement>("values");
  const targetEl = getElement<HTMLInputElement>("target");
  const capacityEl = getElement<HTMLInputElement>("capacity");
  const scriptEl = getElement<HTMLTextAreaElement>("script");
  const queueCapacityEl = getElement<HTMLInputElement>("queue-capacity");
  const queueScriptEl = getElement<HTMLTextAreaElement>("queue-script");
  const binarySearchInputsEl = getElement<HTMLElement>("binary-search-inputs");
  const stackInputsEl = getElement<HTMLElement>("stack-inputs");
  const queueInputsEl = getElement<HTMLElement>("queue-inputs");
  const runEl = getElement<HTMLButtonElement>("run");
  const playerEl = getElement<HTMLElement>("player");
  const stepBackEl = getElement<HTMLButtonElement>("step-back");
  const stepForwardEl = getElement<HTMLButtonElement>("step-forward");
  const playPauseEl = getElement<HTMLButtonElement>("play-pause");
  const speedEl = getElement<HTMLInputElement>("speed");
  const speedLabelEl = getElement<HTMLElement>("speed-label");
  const seekEl = getElement<HTMLInputElement>("seek");
  const frameLabelEl = getElement<HTMLElement>("frame-label");

  // Показываем поля ввода под выбранный алгоритм: у каждого свой блок.
  const syncInputs = (): void => {
    const algorithm = algorithmEl.value;
    binarySearchInputsEl.hidden = algorithm !== "binary-search";
    stackInputsEl.hidden = algorithm !== "stack";
    queueInputsEl.hidden = algorithm !== "queue";
  };
  algorithmEl.addEventListener("change", syncInputs);
  syncInputs();

  const player = createPlayer([]);
  let stream = createStream({ url: "/api/run", player });

  const updatePlayerControls = (): void => {
    const total = player.framesCount();
    const index = player.currentIndex();
    seekEl.min = "1";
    seekEl.max = String(Math.max(1, total));
    seekEl.value = String(index + 1);
    frameLabelEl.textContent = total === 0 ? "—" : `${index + 1} / ${total}`;
    playPauseEl.textContent = player.isPlaying() ? "⏸ Пауза" : "▶ Старт";
    stepBackEl.disabled = index === 0;
    stepForwardEl.disabled = total === 0 || index >= total - 1;
  };

  const render = (frame: Frame): void => {
    if (renderFrame(frame, explanationEl, arrayEl)) {
      playerEl.hidden = false;
      updatePlayerControls();
    }
  };

  player.subscribe(render);

  runEl.addEventListener("click", () => {
    try {
      const algorithm = algorithmEl.value as Algorithm;
      if (!ALGORITHMS.includes(algorithm)) {
        explanationEl.textContent = `алгоритм «${algorithm}» пока не подключён`;
        return;
      }
      // Вход проверяем до Run; URL собираем под алгоритм.
      let url: string;
      if (algorithm === "stack") {
        const capacity = parseStackCapacity(capacityEl.value);
        parseStackScenario(scriptEl.value, capacity); // валидация; на сервер уедет сырой текст
        url = buildStackRunURL(capacity, scriptEl.value);
      } else if (algorithm === "queue") {
        const capacity = parseQueueCapacity(queueCapacityEl.value);
        parseQueueScenario(queueScriptEl.value, capacity); // валидация; на сервер уедет сырой текст
        url = buildQueueRunURL(capacity, queueScriptEl.value);
      } else {
        const values = parseValues(valuesEl.value);
        const target = parseTarget(targetEl.value);
        url = buildRunURL(algorithm, values, target);
      }

      stream.stop();
      player.load([]); // новый Run — сбрасываем кадры, чтобы не смешивать с предыдущим
      playerEl.hidden = true;
      explanationEl.textContent = "Запрашиваю кадры у сервера…";
      arrayEl.replaceChildren();
      stream = createStream({
        url,
        player,
        sink: {
          onFrame: () => updatePlayerControls(),
          onError: (msg) => {
            explanationEl.textContent = `Ошибка: ${msg}`;
          },
          onEnd: () => updatePlayerControls(),
        },
      });
      void stream.start();
    } catch (err) {
      explanationEl.textContent = `Ошибка ввода: ${(err as Error).message}`;
    }
  });

  stepBackEl.addEventListener("click", () => player.stepBackward());
  stepForwardEl.addEventListener("click", () => player.stepForward());
  playPauseEl.addEventListener("click", () => {
    if (player.isPlaying()) player.pause();
    else player.play();
  });

  speedEl.addEventListener("input", () => {
    const value = Number(speedEl.value);
    player.setSpeed(value);
    speedLabelEl.textContent = `${value}×`;
  });
  // Начальное значение скорости синхронизируем с плеером.
  {
    const v = Number(speedEl.value);
    player.setSpeed(v);
    speedLabelEl.textContent = `${v}×`;
  }

  seekEl.addEventListener("input", () => {
    player.seek(Number(seekEl.value) - 1);
  });
}

init();