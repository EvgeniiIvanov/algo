// Общая грамматика десятичного целого для клиентского парсинга входа
// (сценарий стека в scenario.ts и вход бинарного поиска в binarysearch.ts) —
// зеркало Go strconv.Atoi: необязательный знак, только десятичные цифры,
// ведущие нули допустимы. Единственное место с regex ^[+-]?\d+$, Number()
// и Number.isSafeInteger; callers адаптируют результат к своим сообщениям,
// а грамматику не размножают. Полного int64-паритета с сервером нет
// осознанно: Number не представляет целые за Number.MAX_SAFE_INTEGER
// точно, поэтому такие значения клиент отвергает сам, а не молча
// округляет (результаты ревью тикетов 08 и 13).

/**
 * Результат разбора десятичного целого: успех или причина отказа —
 * некорректный синтаксис ("syntax") либо значение вне безопасного целого
 * ("unsafe-range"). В input — обрезанная строка, чтобы caller показал в
 * сообщении ровно то, что разбирал парсер.
 */
export type DecimalIntResult =
  | { ok: true; value: number }
  | { ok: false; reason: "syntax" | "unsafe-range"; input: string };

/**
 * Разобрать строку как десятичное целое: парсер сам делает trim, а дальше
 * грамматика Go strconv.Atoi. Пустая строка, дробные, экспонента,
 * 0x-нотация и мусор — синтаксическая ошибка; целое за пределами
 * [-Number.MAX_SAFE_INTEGER, +Number.MAX_SAFE_INTEGER] — unsafe-range.
 */
export function parseDecimalInteger(raw: string): DecimalIntResult {
  const input = raw.trim();
  if (!/^[+-]?\d+$/.test(input)) {
    return { ok: false, reason: "syntax", input };
  }
  const n = Number(input);
  // Go int64 переполняется и даёт ошибку парсера; на клиенте граница — Number.MAX_SAFE_INTEGER.
  if (!Number.isSafeInteger(n)) {
    return { ok: false, reason: "unsafe-range", input };
  }
  return { ok: true, value: n };
}
