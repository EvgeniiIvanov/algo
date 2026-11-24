// Тест шва фронта (docs/SPEC.md, «Четвёртый шов»): парсинг сценария Trie,
// dry-run лимита живых rune-узлов и построение Run URL — ошибки ввода
// ловятся до сериализации и Run. Зеркало серверной валидации
// (visual/server/trie.go): грамматика строки, один непустой токен из
// валидного UTF-8 (последовательность Unicode scalar values), лимиты
// 500 команд, 100 rune в слове и 100 живых rune-узлов (пиковая семантика).
import { describe, expect, it } from "vitest";
import { buildTrieRunURL } from "../src/sse";
import {
  MAX_TRIE_COMMANDS,
  MAX_TRIE_NODES,
  MAX_TRIE_RUNES,
  checkTrieNodeLimit,
  countRunes,
  hasLoneSurrogate,
  parseTrieScenario,
} from "../src/trie";
import type { TrieCommand } from "../src/trie";

describe("parseTrieScenario — удачный разбор", () => {
  it("разбирает сценарий с insert, search, prefix и delete", () => {
    expect(
      parseTrieScenario("insert кот\nsearch кот\nprefix ко\ndelete ком"),
    ).toEqual([
      { op: "insert", word: "кот" },
      { op: "search", word: "кот" },
      { op: "prefix", word: "ко" },
      { op: "delete", word: "ком" },
    ]);
  });

  it("пропускает пустые строки и строки только из пробелов", () => {
    expect(parseTrieScenario("\n\ninsert кот\n   \n\t\ndelete ком\n\n")).toEqual([
      { op: "insert", word: "кот" },
      { op: "delete", word: "ком" },
    ]);
    expect(parseTrieScenario("  search   ком  ")).toEqual([
      { op: "search", word: "ком" },
    ]);
  });

  it("повторная вставка и промахи search/prefix/delete — легальные шаги", () => {
    expect(() =>
      parseTrieScenario("insert кот\ninsert кот\nsearch ком\nprefix х\ndelete ком"),
    ).not.toThrow();
  });

  it("аргумент — любой непустой токен без пробелов, включая пунктуацию", () => {
    expect(parseTrieScenario("insert он-лайн_v2!")).toEqual([
      { op: "insert", word: "он-лайн_v2!" },
    ]);
  });
});

// Паритет токенизации с сервером (только парсер Trie): whitespace — по
// unicode.IsSpace (scenarioFields.ts), как в strings.Fields на сервере.
// BOM U+FEFF — не разделитель ни на клиенте, ни на сервере; NEL U+0085 —
// разделитель. Исторические парсеры (scenario.ts) держат прежнюю
// JS-токенизацию — их регрессии в scenarioFields.test.ts и scenario.test.ts.
describe("parseTrieScenario — whitespace-паритет с сервером", () => {
  it("NBSP U+00A0 и идеографический пробел U+3000 разделяют действие и аргумент", () => {
    expect(parseTrieScenario("insert\u00A0кот")).toEqual([
      { op: "insert", word: "кот" },
    ]);
    expect(parseTrieScenario("insert\u3000кот")).toEqual([
      { op: "insert", word: "кот" },
    ]);
  });

  it("CRLF между командами корректен: \\r не приклеивается к токенам", () => {
    expect(parseTrieScenario("insert кот\r\ndelete кот")).toEqual([
      { op: "insert", word: "кот" },
      { op: "delete", word: "кот" },
    ]);
  });

  it("строка только из Unicode whitespace игнорируется", () => {
    expect(parseTrieScenario("\u00A0\u3000\ninsert кот\n\u3000")).toEqual([
      { op: "insert", word: "кот" },
    ]);
  });

  it("BOM-prefixed команда отвергается до Run: U+FEFF — не whitespace в Go", () => {
    // Сервер видит команду «\uFEFFinsert» и отвергает сценарий; фронт
    // повторяет это решение до Run, а не молча отбрасывает BOM.
    expect(() => parseTrieScenario("\uFEFFinsert кот")).toThrow(
      /строка 1: неизвестная команда «\uFEFFinsert»/,
    );
    // Токен с BOM внутри: verb — вся склейка «insert\uFEFFкот» — неизвестная
    // команда (и на сервере Go строка разобрана той же семантикой).
    expect(() => parseTrieScenario("insert\uFEFFкот")).toThrow(
      /строка 1: неизвестная команда «insert\uFEFFкот»/,
    );
  });

  it("NEL U+0085 — разделитель: аргумент разобран как отдельный токен", () => {
    // В Go U+0085 пробельный (в JS \s его нет): сервер принял бы
    // «insert U+0085 кот» как insert кот; фронт зеркалит это.
    expect(parseTrieScenario("insert\u0085кот")).toEqual([
      { op: "insert", word: "кот" },
    ]);
  });
});

describe("parseTrieScenario — ошибки ввода до Run", () => {
  it("отвергает неизвестную команду с номером строки", () => {
    expect(() => parseTrieScenario("insert кот\nupsert кот")).toThrow(
      /строка 2.*неизвестная команда «upsert»/,
    );
  });

  it("отвергает команду без аргумента", () => {
    expect(() => parseTrieScenario("insert")).toThrow(
      /команда «insert»: не хватает аргумента/,
    );
    expect(() => parseTrieScenario("delete кот\nprefix")).toThrow(
      /строка 2: команда «prefix»: не хватает аргумента/,
    );
  });

  it("отвергает лишние аргументы", () => {
    expect(() => parseTrieScenario("insert кот код")).toThrow(
      /лишние аргументы/,
    );
    expect(() => parseTrieScenario("search кот код при")).toThrow(
      /лишние аргументы/,
    );
  });

  it("отвергает пустой сценарий (только пустые строки и пробелы)", () => {
    expect(() => parseTrieScenario("")).toThrow(/сценарий пуст/);
    expect(() => parseTrieScenario("\n \n\t\n")).toThrow(/сценарий пуст/);
  });
});

describe("parseTrieScenario — одиночная UTF-16 surrogate до сериализации", () => {
  it("отвергает верхнюю половину без нижней", () => {
    expect(() => parseTrieScenario(`insert кот\uD800`)).toThrow(
      /одиночная UTF-16 surrogate не является Unicode scalar value/,
    );
  });

  it("отвергает нижнюю половину без верхней", () => {
    expect(() => parseTrieScenario(`delete \uDC00`)).toThrow(
      /одиночная UTF-16 surrogate/,
    );
  });

  it("парная surrogate (символ вне BMP) принимается", () => {
    expect(parseTrieScenario("insert 😀")).toEqual([{ op: "insert", word: "😀" }]);
  });

  it("hasLoneSurrogate различает одиночную и парную surrogate", () => {
    expect(hasLoneSurrogate("\uD800")).toBe(true);
    expect(hasLoneSurrogate("\uDC00")).toBe(true);
    expect(hasLoneSurrogate("кот\uD800кот")).toBe(true);
    expect(hasLoneSurrogate("\uD800x")).toBe(true);
    expect(hasLoneSurrogate("😀")).toBe(false);
    expect(hasLoneSurrogate("кот")).toBe(false);
  });
});

describe("подсчёт rune — по Unicode scalar values, не UTF-16 code units", () => {
  it("😀 — одна руна, хотя занимает пару UTF-16 code units", () => {
    expect(countRunes("😀")).toBe(1);
    expect("😀".length).toBe(2); // code units — для контраста
    expect(countRunes("кот")).toBe(3);
  });

  it("аргумент ровно из 100 😀 принимается", () => {
    const word = "😀".repeat(MAX_TRIE_RUNES);
    expect(word.length).toBe(200); // 200 code units, но 100 rune
    expect(parseTrieScenario(`search ${word}`)).toEqual([
      { op: "search", word },
    ]);
  });

  it("аргумент из 101 😀 отвергается до Run", () => {
    const word = "😀".repeat(MAX_TRIE_RUNES + 1);
    expect(() => parseTrieScenario(`search ${word}`)).toThrow(
      new RegExp(`длиннее ${MAX_TRIE_RUNES} rune.*длина ${MAX_TRIE_RUNES + 1}`),
    );
  });

  it("смешанная длина считается по рунам: 99 латинских + 😀 — ровно 100", () => {
    const word = "a".repeat(MAX_TRIE_RUNES - 1) + "😀";
    expect(word.length).toBe(MAX_TRIE_RUNES + 1); // 101 code unit
    expect(() => parseTrieScenario(`insert ${word}`)).not.toThrow();
  });

  it("аргумент из 101 обычной руны отвергается", () => {
    expect(() => parseTrieScenario(`insert ${"к".repeat(MAX_TRIE_RUNES + 1)}`)).toThrow(
      /длиннее 100 rune/,
    );
  });

  it("очень длинный токен отсекается без подсчёта и без эха аргумента", () => {
    expect(() => parseTrieScenario(`insert ${"к".repeat(5000)}`)).toThrow(
      /сократите аргумент/,
    );
  });
});

describe("parseTrieScenario — лимит команд", () => {
  const script = (n: number) =>
    Array.from({ length: n }, (_, i) => `search ${i}`).join("\n");

  it("ровно 500 команд принимается", () => {
    expect(parseTrieScenario(script(MAX_TRIE_COMMANDS))).toHaveLength(
      MAX_TRIE_COMMANDS,
    );
  });

  it("501-я команда отвергается с номером первой лишней строки", () => {
    expect(() => parseTrieScenario(script(MAX_TRIE_COMMANDS + 1))).toThrow(
      new RegExp(
        `слишком длинный.*не более ${MAX_TRIE_COMMANDS} команд.*на строке ${MAX_TRIE_COMMANDS + 1}`,
      ),
    );
  });

  it("лимит считается по непустым командам: пустые строки не занимают места", () => {
    const scriptWithBlanks = "\n" + script(MAX_TRIE_COMMANDS) + "\n  \n";
    expect(() => parseTrieScenario(scriptWithBlanks)).not.toThrow();
  });
});

describe("checkTrieNodeLimit — dry-run пикового числа живых rune-узлов", () => {
  const word = (ch: string, n: number) => ch.repeat(n);

  it("вставка ровно 100 rune создаёт 100 живых узлов и принимается", () => {
    expect(() => checkTrieNodeLimit([{ op: "insert", word: word("к", MAX_TRIE_NODES) }]))
      .not.toThrow();
  });

  it("101-й узел отвергается до Run с номером команды", () => {
    expect(() =>
      checkTrieNodeLimit([
        { op: "insert", word: word("к", MAX_TRIE_NODES) },
        { op: "insert", word: "б" },
      ]),
    ).toThrow(
      new RegExp(
        `лимит живых rune-узлов.*не более ${MAX_TRIE_NODES}.*команда №2 \\(insert б\\) даёт ${MAX_TRIE_NODES + 1}`,
      ),
    );
  });

  it("превышение пика отвергается, даже если финал укладывается в лимит", () => {
    const script: TrieCommand[] = [
      { op: "insert", word: word("а", 60) },
      { op: "insert", word: word("б", 60) }, // пик 120
      { op: "delete", word: word("а", 60) },
      { op: "delete", word: word("б", 60) }, // финал 0
    ];
    expect(() => checkTrieNodeLimit(script)).toThrow(/команда №2.*даёт 120/);
  });

  it("удаление с pruning освобождает место под новые вставки", () => {
    const script: TrieCommand[] = [
      { op: "insert", word: word("к", MAX_TRIE_NODES) },
      { op: "delete", word: word("к", MAX_TRIE_NODES) },
      { op: "insert", word: word("м", MAX_TRIE_NODES) },
    ];
    expect(() => checkTrieNodeLimit(script)).not.toThrow();
  });

  it("повторная вставка не растит дерево", () => {
    const repeat: TrieCommand[] = Array.from({ length: MAX_TRIE_NODES + 50 }, () => ({
      op: "insert",
      word: "кот",
    }));
    expect(() => checkTrieNodeLimit(repeat)).not.toThrow();
  });

  it("промахи search/prefix/delete и промах удаления не меняют счётчик", () => {
    const script: TrieCommand[] = [
      { op: "insert", word: word("к", MAX_TRIE_NODES) },
      { op: "search", word: word("м", 50) },
      { op: "prefix", word: word("м", 50) },
      { op: "delete", word: word("м", 50) }, // промах по отсутствующему ребру
      { op: "delete", word: word("к", MAX_TRIE_NODES - 1) }, // путь есть, не конец слова
    ];
    expect(() => checkTrieNodeLimit(script)).not.toThrow();
  });

  it("после не-удалившегося промаха свободного места нет", () => {
    const script: TrieCommand[] = [
      { op: "insert", word: word("к", MAX_TRIE_NODES) },
      { op: "delete", word: word("м", MAX_TRIE_NODES) }, // промах: pruning нет
      { op: "insert", word: "б" },
    ];
    expect(() => checkTrieNodeLimit(script)).toThrow(/команда №3.*даёт 101/);
  });

  it("поиск ровно 100 😀 в пустом Trie: dry-run даёт ровно ноль живых узлов", () => {
    const commands = parseTrieScenario(`search ${"😀".repeat(MAX_TRIE_RUNES)}`);
    // Не «просто не упали по лимиту»: события проверки ребра не создают
    // узлов — живых узлов ноль, и результат dry-run это показывает.
    expect(checkTrieNodeLimit(commands)).toBe(0);
  });

  it("dry-run возвращает пиковое число живых узлов: 100 вставленных рун — ровно 100", () => {
    expect(
      checkTrieNodeLimit([{ op: "insert", word: "к".repeat(MAX_TRIE_NODES) }]),
    ).toBe(MAX_TRIE_NODES);
  });

  it("пик против финала: вставка 60 рун и удаление — пик 60, финал 0", () => {
    // Вставили 60 рун (пик 60), удалили: pruning срезает все 60 узлов,
    // финальное живое число — 0. Возвращаемый пик берётся по максимуму
    // после каждой команды и не уменьшается удалением.
    expect(
      checkTrieNodeLimit([
        { op: "insert", word: "к".repeat(60) },
        { op: "delete", word: "к".repeat(60) },
      ]),
    ).toBe(60);
    // Финал действительно 0, а не 60: следующая вставка ещё 50 рун не
    // отвергается (при неочищенных 60 было бы 110 > 100 — Error), а пик
    // остаётся прежним максимумом 60.
    expect(
      checkTrieNodeLimit([
        { op: "insert", word: "к".repeat(60) },
        { op: "delete", word: "к".repeat(60) },
        { op: "insert", word: "м".repeat(50) },
      ]),
    ).toBe(60);
  });
});

describe("buildTrieRunURL", () => {
  it("строит URL маршрута /api/run/trie с параметром script", () => {
    // URLSearchParams кодирует кириллицу процентами, перенос строки — %0A.
    expect(buildTrieRunURL("insert кот")).toBe(
      "/api/run/trie?script=insert+%D0%BA%D0%BE%D1%82",
    );
    expect(buildTrieRunURL("insert кот\nsearch ком")).toBe(
      "/api/run/trie?script=insert+%D0%BA%D0%BE%D1%82%0Asearch+%D0%BA%D0%BE%D0%BC",
    );
  });

  it("символ вне BMP едет одной кодовой точкой UTF-8", () => {
    expect(buildTrieRunURL("insert 😀")).toBe(
      "/api/run/trie?script=insert+%F0%9F%98%80",
    );
  });
});
