// Парсинг входа бинарного поиска на клиенте (зеркало visual/server/server.go):
// ловим ошибки ввода до Run; сервер перепроверяет — он авторитетен.
// Формы, которые Number() принимает, а серверный strconv.Atoi отвергнет
// (2.0, 1e2, 0x10), сыплются здесь понятной ошибкой, а не HTTP 400 после Run.
// Грамматика значений — Go strconv.Atoi, см. общий модуль decimalInteger.ts:
// там же живёт ограничение безопасного целого (Number.MAX_SAFE_INTEGER),
// поэтому значения за его пределами клиент отвергает сам, а не молча
// округляет (верхняя граница int на сервере шире — неустранимое
// ограничение клиента).

import { parseDecimalInteger } from "./decimalInteger";

/**
 * Разобрать одно целое по общей грамматике decimalInteger.ts
 * (Go strconv.Atoi в пределах безопасного целого) и бросить Error
 * с контекстным сообщением: при синтаксической ошибке — с подсказкой
 * поля fieldHint (в стиле сервера), при выходе за безопасное целое —
 * с сообщением именно про диапазон, а не «не целое».
 */
export function parseBinaryInt(raw: string, fieldHint: string): number {
  const result = parseDecimalInteger(raw);
  if (!result.ok) {
    if (result.reason === "unsafe-range") {
      throw new Error(
        `«${result.input}» — за пределами точного диапазона JavaScript: поддерживаются целые от -${Number.MAX_SAFE_INTEGER} до ${Number.MAX_SAFE_INTEGER}`,
      );
    }
    throw new Error(`не удалось разобрать «${result.input}» как целое число: ${fieldHint}`);
  }
  return result.value;
}

/**
 * Разобрать values — целые числа через запятую. Пустой ввод и пустой
 * компонент между запятыми — ошибки: сервер их тоже отвергнет.
 * Пробелы вокруг числа допускаются, как на сервере (TrimSpace).
 */
export function parseValues(raw: string): number[] {
  const parts = raw.split(",").map((p) => p.trim());
  if (parts.length === 1 && parts[0] === "") {
    throw new Error("введите массив — целые числа через запятую");
  }
  return parts.map((p) => parseBinaryInt(p, "values — целые числа через запятую"));
}

/**
 * Разобрать target — одно целое число. Пустая строка — ошибка входа:
 * пустой ввод не превращается в 0 (как было бы при Number(raw)).
 */
export function parseTarget(raw: string): number {
  if (raw.trim() === "") {
    throw new Error("введите искомое значение — целое число");
  }
  return parseBinaryInt(raw, "target — одно целое число");
}
