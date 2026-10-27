// Парсинг входа обхода бинарного дерева на клиенте (зеркало
// visual/server/tree.go): ловим ошибки ввода до Run; сервер перепроверяет —
// он авторитетен. Вложенный JSON дерева (value, left, right): отсутствующий
// или null потомок — пустая ветвь, верхний null — пустое дерево.
// Лимит узлов — MAX_TREE_NODES, как на сервере.

/** Ограничения входа — как на сервере (visual/server/tree.go). */
export const MAX_TREE_NODES = 100;

/**
 * Верхняя граница длины JSON-числа в position value — как на сервере
 * (maxNumberLiteral в visual/server/tree.go): ни один литерал,
 * помещающийся в целое, не требует и десятой доли этого размера;
 * длиннее — только дробные/нулевые хвосты.
 */
export const MAX_NUMBER_LITERAL = 4096;

/** Порядки обхода — как на сервере (render.TreeOrder*). */
export const TREE_ORDERS = ["in-order", "pre-order", "post-order"] as const;
export type TreeOrder = (typeof TREE_ORDERS)[number];

/** Страж типа: выбран порядок обхода дерева. */
export function isTreeOrder(value: string): value is TreeOrder {
  return (TREE_ORDERS as readonly string[]).includes(value);
}

/** Разобрать порядок обхода: бросает Error для неизвестного или пустого. */
export function parseTreeOrder(raw: string): TreeOrder {
  const trimmed = raw.trim();
  if (trimmed === "") {
    throw new Error("выберите порядок обхода: in-order, pre-order или post-order");
  }
  if (!isTreeOrder(trimmed)) {
    throw new Error(
      `неизвестный порядок обхода «${trimmed}»: ожидается in-order, pre-order или post-order`,
    );
  }
  return trimmed;
}

/**
 * Проверить вложенный JSON дерева (зеркало parseTreeJSON сервера):
 * верхний null — пустое дерево; каждый узел — объект {value, left, right}
 * с целым значением; отсутствующий или null потомок — пустая ветвь;
 * больше MAX_TREE_NODES узлов — ошибка. Бросает Error с понятным
 * сообщением. Числа за пределами точного диапазона JavaScript клиент
 * отвергает сам, а не молча округляет (см. decimalInteger.ts).
 */
export function parseTreeInput(raw: string): void {
  if (raw.trim() === "") {
    throw new Error(
      "введите дерево — вложенный JSON, например {\"value\":1,\"left\":{\"value\":2}}; null — пустое дерево",
    );
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch (err) {
    throw new Error(`не удалось разобрать JSON дерева: ${(err as Error).message}`, { cause: err });
  }
  // Точная проверка value-литералов по сырой строке: JSON.parse уже
  // округлил числа в Number, и нецелая дробь (1.000…0001) дошла бы до
  // проверки как 1 (см. checkValueLiteral). Экранированный ключ
  // («\u0076alue») в литералах учитывается декодером строк сканера.
  checkValueLiterals(raw);
  buildTreeInput(parsed, "", 0);
}

/**
 * Рекурсивная проверка узла (зеркало buildTree на сервере): value —
 * целое в пределах безопасного диапазона, потомки — вложенные узлы
 * или null/без ключа. count — число узлов выше по дереву; путь —
 * символы L/R от корня ("" — корень) для сообщений об ошибках.
 * Возвращает число узлов в поддереве.
 */
function buildTreeInput(parsed: unknown, branch: string, count: number): number {
  if (parsed === null) {
    return 0; // верхний null или null-потомок — пустая ветвь
  }
  if (typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error(
      `узел дерева${treeContext(branch)} должен быть объектом {value, left, right} или null`,
    );
  }
  const obj = parsed as Record<string, unknown>;
  const value = obj.value;
  if (typeof value !== "number" || !Number.isInteger(value)) {
    throw new Error(
      `значение узла${treeContext(branch)} не целое: value — целое число`,
    );
  }
  if (Math.abs(value) > Number.MAX_SAFE_INTEGER) {
    throw new Error(
      `значение узла${treeContext(branch)} за пределами точного диапазона JavaScript: поддерживаются целые от -${Number.MAX_SAFE_INTEGER} до ${Number.MAX_SAFE_INTEGER}`,
    );
  }
  if (count + 1 > MAX_TREE_NODES) {
    throw new Error(
      `дерево слишком большое: разрешено не более ${MAX_TREE_NODES} узлов; сократите вход`,
    );
  }
  // Отсутствующий ключ и явный null равнозначны — оба пустая ветвь.
  const leftCount =
    obj.left === undefined || obj.left === null
      ? 0
      : buildTreeInput(obj.left, branch + "L", count + 1);
  const rightCount =
    obj.right === undefined || obj.right === null
      ? 0
      : buildTreeInput(obj.right, branch + "R", count + 1 + leftCount);
  return 1 + leftCount + rightCount;
}

/** Сегмент сообщения об ошибке: путь узла от корня ("" — корень). */
function treeContext(branch: string): string {
  return branch === "" ? " (корень)" : ` (путь ${branch})`;
}

/**
 * Точная проверка value-литералов (зеркало treeValue на сервере) по сырой
 * строке, до преобразования в Number: JSON.parse уже округлил числа, и
 * нецелая дробь вида 1.000…0001 (дробный хвост после 47 нулей) доходит до
 * проверки как 1 — Number.isInteger её пропустил бы. Сервер отвергает
 * такое значение 400-кой, поэтому клиент обязан проверить литерал тем же
 * способом: точная нецелость считается по счёту цифр и показателя
 * (см. checkValueLiteral). Экранированный ключ («\u0076alue») сканер
 * раскрывает тем же декодером строк, что JSON.parse, — расхождений по
 * именам ключей нет; посторонние поля и вложенные структуры сервер не
 * смотрит — их сканер только корректно пропускает (строки с
 * экранированием, вложенные скобки и массивы).
 */

/**
 * Ошибка внутреннего сканера: JSON.parse уже доказал валидность грамматики,
 * поэтому любые несовпадения — ошибка сканера, а не входа. Чтобы
 * не отвергать корректный ввод из-за несовершенной внутренней проверки,
 * сканер на них молча останавливается (сервер остаётся авторитетен и
 * перепроверяет то же самое).
 */
class ScanAbort extends Error {}

/**
 * Класс контекста сканера: «объект-узел» — объект, который сервер считает
 * узлом дерева: верхний уровень и значения ключей left/right в таких
 * объектах. Числа в поле value объектов-узлов проверяются точно; числа
 * в посторонних полях и вложенных структурах сервер не смотрит — их
 * сканер только пропускает (строки с экранированием, вложенные скобки).
 */
function checkValueLiterals(raw: string): void {
  try {
    let i = 0;
    const len = raw.length;

    /** Отказ сканера: не вход виноват, а несовершенство внутренней проверки. */
    function abort(): never {
      throw new ScanAbort("сканер не смог завершить проверку литералов");
    }

    const skipWhitespace = (): void => {
      while (
        i < len &&
        (raw[i] === " " || raw[i] === "\t" || raw[i] === "\n" || raw[i] === "\r")
      ) {
        i++;
      }
    };

    /** Пропустить строку с учётом экранирования; возвращает её значение. */
    const skipString = (collect: boolean): string => {
      if (raw[i] !== '"' || i >= len) abort();
      i++;
      let out = "";
      while (i < len) {
        const ch = raw[i];
        if (ch === "\\") {
          const esc = raw[i + 1];
          if (esc === undefined) abort();
          const pairs: Record<string, string> = {
            '"': '"', "\\": "\\", "/": "/", b: "\b", f: "\f",
            n: "\n", r: "\r", t: "\t",
          };
          if (esc in pairs) {
            if (collect) out += pairs[esc];
            i += 2;
          } else if (esc === "u") {
            const hex = raw.slice(i + 2, i + 6);
            if (!/^[0-9a-fA-F]{4}$/.test(hex)) abort();
            if (collect) out += String.fromCharCode(parseInt(hex, 16));
            i += 6;
          } else {
            abort();
          }
          continue;
        }
        if (ch === '"') {
          i++;
          return out;
        }
        if (collect) out += ch;
        i++;
      }
      abort();
    };

    /** Пропустить литерал JSON-числа: [-] int[.frac][eE[±]digits]. */
    const skipNumber = (collect: boolean): string => {
      const start = i;
      while (i < len) {
        const ch = raw[i];
        if (
          (ch >= "0" && ch <= "9") || ch === "-" || ch === "+" ||
          ch === "." || ch === "e" || ch === "E"
        ) {
          i++;
          continue;
        }
        break;
      }
      if (collect) return raw.slice(start, i);
      return "";
    };

    /** Пропуск произвольного значения (не узла дерева). */
    const skipValue = (): void => {
      skipWhitespace();
      if (i >= len) abort();
      const ch = raw[i];
      if (ch === '"') {
        skipString(false);
        return;
      }
      if (ch === "[") {
        i++;
        skipWhitespace();
        if (raw[i] === "]") {
          i++;
          return;
        }
        for (;;) {
          skipValue();
          skipWhitespace();
          const c = raw[i];
          if (c === ",") {
            i++;
            continue;
          }
          if (c === "]") {
            i++;
            return;
          }
          abort();
        }
      }
      if (ch === "{") {
        skipObject(false, "");
        return;
      }
      if ((ch >= "0" && ch <= "9") || ch === "-") {
        skipNumber(false);
        return;
      }
      skipKeywordStrict();
    };

    /** Пропуск true/false/null. */
    const skipKeywordStrict = (): void => {
      skipWhitespace();
      if (
        raw.startsWith("true", i) ||
        raw.startsWith("false", i) ||
        raw.startsWith("null", i)
      ) {
        i += raw[i] === "n" ? 4 : 5;
        return;
      }
      abort();
    };

    /**
     * Обойти объект. В объектах-узлах (node=true): значения left/right —
     * тоже узлы; ключ «value» в последней записи с этим ключом, если это
     * число, проходит строгую проверку литерала (как JSON.parse, так и
     * сервер оставляют при повторении ключа последнюю запись).
     */
    const skipObject = (node: boolean, path: string): void => {
      skipWhitespace();
      if (raw[i] !== "{") abort();
      i++;
      skipWhitespace();
      if (raw[i] === "}") {
        i++;
        return;
      }
      let valueLit: string | null = null;
      let valueIsNumber = false;
      for (;;) {
        skipWhitespace();
        const key = skipString(true);
        skipWhitespace();
        if (raw[i] !== ":") abort();
        i++;
        skipWhitespace();
        const ch = raw[i];
        const isNumber = ch === "-" || (ch >= "0" && ch <= "9");
        if (node && key === "value") {
          if (isNumber) {
            valueLit = skipNumber(true);
            valueIsNumber = true;
          } else {
            valueLit = null;
            valueIsNumber = false;
            skipValue();
          }
        } else if (node && (key === "left" || key === "right")) {
          const branchPath = path + (key === "left" ? "L" : "R");
          skipWhitespace();
          if (raw[i] === "{") {
            skipObject(true, branchPath);
          } else {
            skipValue();
          }
        } else {
          skipValue();
        }
        skipWhitespace();
        const c = raw[i];
        if (c === ",") {
          i++;
          continue;
        }
        if (c === "}") {
          i++;
          if (valueIsNumber && valueLit !== null) {
            checkValueLiteral(valueLit, treeContext(path));
          }
          return;
        }
        abort();
      }
    };

    // верхний уровень — узел дерева: null — пустое дерево, число/строка —
    // ошибка «должен быть объектом» в buildTreeInput.
    skipWhitespace();
    if (raw[i] === "{") {
      skipObject(true, "");
    } else {
      skipValue();
    }
  } catch (err) {
    if (err instanceof ScanAbort) return; // внутренняя ошибка: сервер перепроверит
    throw err;
  }
}

/**
 * Точная проверка одного литерала как целого (порт treeValue из
 * visual/server/tree.go; вместо int64 — граница Number.MAX_SAFE_INTEGER,
 * в духе decimalInteger.ts): нецелость определяется счётом значащих цифр
 * и показателя — без float64 и BigInt, которые сами округляют длинные
 * записи.
 */
function checkValueLiteral(lit: string, ctx: string): void {
  if (lit.length > MAX_NUMBER_LITERAL) {
    throw new Error(
      `значение узла${ctx} слишком длинное (лимит ${MAX_NUMBER_LITERAL} символов): value — целое число`,
    );
  }

  // Структура литерала: [-] цифры [. цифры] [eE [+-] цифры].
  const neg = lit.startsWith("-");
  const body = neg ? lit.slice(1) : lit;
  const eIdx = Math.min(
    ...["e" as const, "E" as const]
      .map((ch) => body.indexOf(ch))
      .filter((idx) => idx >= 0),
  );
  const mant = eIdx >= 0 ? body.slice(0, eIdx) : body;
  const es = eIdx >= 0 ? body.slice(eIdx + 1) : "";
  const dotIdx = mant.indexOf(".");
  const intPart = dotIdx >= 0 ? mant.slice(0, dotIdx) : mant;
  const frac = dotIdx >= 0 ? mant.slice(dotIdx + 1) : "";

  // Значащие цифры (ведущие нули не считаются): «0», «0.0» — нуль при
  // любом показателе (зеркало сервера).
  let digits = (intPart + frac).replace(/^0+/, "");
  if (digits === "") return; // нуль: Number(0) — целое, проверка пройдена

  // Показатель: Number(es) конечен или Infinity — NaN невозможен,
  // грамматику экспоненты уже гарантировал JSON.parse. Границы ниже
  // сравниваются с exp без вычитаний, поэтому и Infinity проходит
  // корректно.
  let exp = 0;
  if (es !== "") {
    exp = Number(es);
    // Переполнобезопасные границы — порт сервера (visual/server/tree.go):
    //   сверху exp > 19+len(frac) — после домножения больше 19 значащих
    //   цифр (≥ 10^20), это заведомо за Number.MAX_SAFE_INTEGER;
    //   снизу exp < -(len(frac)+len(digits)) — дробная часть целой быть
    //   не может (k > числа замыкающих нулей t ≤ len(digits)).
    // Границы сравниваются без вычислений над exp, так что работаем и на
    // MaxInt/MinInt-подобных показателях, и на Infinity.
    if (exp > 19 + frac.length) {
      throw new Error(
        `значение узла${ctx} за пределами точного диапазона JavaScript: поддерживаются целые от -${Number.MAX_SAFE_INTEGER} до ${Number.MAX_SAFE_INTEGER}`,
      );
    }
    if (exp < -(frac.length + digits.length)) {
      throw new Error(`значение узла${ctx} не целое: value — целое число`);
    }
  }

  // Замыкющие нули значащих цифр покрывают дробную часть.
  let t = 0;
  while (t < digits.length && digits[digits.length - 1 - t] === "0") t++;
  let E = exp - frac.length;
  if (E < 0) {
    const k = -E;
    if (k > t) {
      throw new Error(`значение узла${ctx} не целое: value — целое число`);
    }
    digits = digits.slice(0, digits.length - k);
    E = 0;
  }
  // Домножение на степени десяти: MAX_SAFE_INTEGER — 16 цифр, поэтому
  // больше — заведомо за пределами безопасного диапазона.
  if (digits.length + E > 16) {
    throw new Error(
      `значение узла${ctx} за пределами точного диапазона JavaScript: поддерживаются целые от -${Number.MAX_SAFE_INTEGER} до ${Number.MAX_SAFE_INTEGER}`,
    );
  }
  digits += "0".repeat(E);
  // Сравнение по строке цифр: Number здесь уже не точен (длина ≤ 16 —
  // на границе 2^53), а правило то же, что в buildTreeInput:
  // |value| ≤ Number.MAX_SAFE_INTEGER = 9007199254740991.
  if (digits.length === 16 && digits > "9007199254740991") {
    throw new Error(
      `значение узла${ctx} за пределами точного диапазона JavaScript: поддерживаются целые от -${Number.MAX_SAFE_INTEGER} до ${Number.MAX_SAFE_INTEGER}`,
    );
  }
}
