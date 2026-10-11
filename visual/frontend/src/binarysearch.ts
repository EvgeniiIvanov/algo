// Парсинг входа бинарного поиска на клиенте (зеркало visual/server/server.go):
// ловим ошибки ввода до Run; сервер перепроверяет — он авторитетен.
// Формы, которые Number() принимает, а серверный strconv.Atoi отвергнет
// (2.0, 1e2, 0x10), сыплются здесь понятной ошибкой, а не HTTP 400 после Run.
// Грамматика значений — Go strconv.Atoi: необязательный знак, только
// десятичные цифры. Полного int64-паритета с сервером нет осознанно
// (по итогам ревью тикета 08): Number не представляет целые больше
// Number.MAX_SAFE_INTEGER точно, поэтому такие значения клиент отвергает
// сам, а не молча округляет. Верхняя граница int на сервере шире — это
// неустранимое ограничение клиента.

/**
 * Разобрать одно целое по грамматике Go strconv.Atoi (необязательный знак,
 * только десятичные цифры) в пределах безопасного целого. what — подсказка
 * поля в сообщении об ошибке, в стиле сервера.
 */
export function parseBinaryInt(raw: string, what: string): number {
  const trimmed = raw.trim();
  if (!/^[+-]?\d+$/.test(trimmed)) {
    throw new Error(`не удалось разобрать «${trimmed}» как целое число: ${what}`);
  }
  const n = Number(trimmed);
  if (!Number.isSafeInteger(n)) {
    throw new Error(
      `«${trimmed}» — за пределами точного диапазона JavaScript: поддерживаются целые от -${Number.MAX_SAFE_INTEGER} до ${Number.MAX_SAFE_INTEGER}`,
    );
  }
  return n;
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
