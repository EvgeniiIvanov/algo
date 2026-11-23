// Парсинг сценария Trie на клиенте (зеркало visual/server/trie.go):
// ловим ошибки ввода до Run; сервер перепроверяет — он авторитетен.
// Сценарий — по одной команде `insert W`, `search W`, `prefix W`,
// `delete W` на строку; аргумент — один непустой Unicode-токен без
// пробелов из валидного UTF-8 (последовательность Unicode scalar values).
// Токенизация строки — общая со всеми парсерами сценариев
// (scenarioFields.ts): whitespace — по unicode.IsSpace, как на сервере.
// Подсчёт длины — по Unicode scalar values (rune), не по UTF-16 code
// units: 😀 — одна руна, UTF-16-пара не считается за две.
//
// Одиночная UTF-16 surrogate в разобранном аргументе — не Unicode scalar
// value (невалидная последовательность scalar values) — отклоняется здесь,
// до сериализации и Run: encodeURIComponent бросает URIError на одиночную
// surrogate, а URLSearchParams может молча заменить её на U+FFFD — явная
// проверка предшествует обоим кодированиям.
//
// Публичная симуляция сценария (dry-run) зеркалит серверную проверку
// лимита живых rune-узлов (visual/server/trie.go, checkTrieNodeLimit):
// пиковое число живых rune-узлов — максимум по состоянию после каждой
// команды, а не только финальное; корень не считается. Превышение лимита
// и иные нарушения отвергаются до отправки Run.

import { splitScenarioFields } from "./scenarioFields";

/** Ограничения входа — как на сервере (visual/server/trie.go). */
export const MAX_TRIE_COMMANDS = 500;
export const MAX_TRIE_RUNES = 100;
export const MAX_TRIE_NODES = 100;

/** Вид операции сценария Trie. */
export type TrieOpKind = "insert" | "search" | "prefix" | "delete";

/** Одна команда сценария Trie: действие над словом (для prefix — префиксом). */
export interface TrieCommand {
  op: TrieOpKind;
  word: string;
}

const TRIE_OPS: readonly TrieOpKind[] = ["insert", "search", "prefix", "delete"];

/**
 * Есть ли в строке одиночная UTF-16 surrogate (U+D800–U+DFFF, не
 * сопровождающаяся второй половиной пары): такая единица — не Unicode
 * scalar value, аргумент с ней — невалидная последовательность scalar
 * values (в терминах сервера — невалидный UTF-8).
 */
export function hasLoneSurrogate(word: string): boolean {
  for (let i = 0; i < word.length; i++) {
    const unit = word.charCodeAt(i);
    if (unit >= 0xd800 && unit <= 0xdbff) {
      // Верхняя половина пары: одиночная, если за ней не идёт нижняя.
      const next = i + 1 < word.length ? word.charCodeAt(i + 1) : -1;
      if (next < 0xdc00 || next > 0xdfff) return true;
      i++; // парная surrogate — одна руна, шагаем через обе единицы
    } else if (unit >= 0xdc00 && unit <= 0xdfff) {
      return true; // нижняя половина без верхней
    }
  }
  return false;
}

/**
 * Число rune (Unicode scalar values) в слове, а не UTF-16 code units:
 * символ вне BMP (😀) — одна руна, хотя занимает пару code units.
 * Вызов законен только для строк без одиночных surrogate — их отвергает
 * hasLoneSurrogate до подсчёта.
 */
export function countRunes(word: string): number {
  return Array.from(word).length;
}

/**
 * Разобрать сценарий Trie: одна на строку — «insert W», «search W»,
 * «prefix W» или «delete W». Пустые строки и строки только из пробелов
 * игнорируются. Бросает Error с понятным сообщением (зеркало сервера):
 * неизвестная команда, не хватает или лишние аргументы, невалидная
 * последовательность scalar values (одиночная UTF-16 surrogate), лимит
 * 100 rune и 500 команд. Повторная вставка и промахи search/prefix/
 * delete — легальные шаги сценария, а не ошибки ввода.
 */
export function parseTrieScenario(raw: string): TrieCommand[] {
  const commands: TrieCommand[] = [];
  const lines = raw.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const fields = splitScenarioFields(lines[i]);
    if (fields.length === 0) continue; // пустая строка — не команда
    const lineNo = i + 1;

    const verb = fields[0];
    if (!(TRIE_OPS as readonly string[]).includes(verb)) {
      throw new Error(
        `строка ${lineNo}: неизвестная команда «${verb}»: ожидается insert, search, prefix или delete`,
      );
    }
    if (fields.length === 1) {
      throw new Error(
        `строка ${lineNo}: команда «${verb}»: не хватает аргумента — ожидается «${verb} W», например ${verb} кот`,
      );
    }
    if (fields.length > 2) {
      throw new Error(
        `строка ${lineNo}: лишние аргументы — ожидается действие и один непустой аргумент, например «${verb} кот»`,
      );
    }
    const word = fields[1];

    // Одиночная UTF-16 surrogate — не Unicode scalar value: отклоняем до
    // сериализации (encodeURIComponent / URLSearchParams) и Run.
    if (hasLoneSurrogate(word)) {
      throw new Error(
        `строка ${lineNo}: аргумент «${word}» — невалидный UTF-8: одиночная UTF-16 surrogate не является Unicode scalar value`,
      );
    }
    // Токен длиннее 4 единиц на rune заведомо длиннее 100 rune — отсекаем
    // сразу, без подсчёта (зеркало серверной границы maxTrieRunes*utf8.UTFMax).
    if (word.length > MAX_TRIE_RUNES * 4) {
      throw new Error(
        `строка ${lineNo}: аргумент длиннее ${MAX_TRIE_RUNES} rune (Unicode scalar values): сократите аргумент`,
      );
    }
    const runes = countRunes(word);
    if (runes > MAX_TRIE_RUNES) {
      throw new Error(
        `аргумент «${word}» на строке ${lineNo} длиннее ${MAX_TRIE_RUNES} rune (Unicode scalar values): длина ${runes}`,
      );
    }
    // Лимит команд проверяется на лету: сценарий-перегрузка не копит
    // команды (как на сервере).
    if (commands.length === MAX_TRIE_COMMANDS) {
      throw new Error(
        `сценарий слишком длинный: разрешено не более ${MAX_TRIE_COMMANDS} команд; первую лишнюю команду видим на строке ${lineNo}`,
      );
    }
    commands.push({ op: verb as TrieOpKind, word });
  }

  if (commands.length === 0) {
    throw new Error(
      "сценарий пуст: пустые строки игнорируются, нужен хотя бы один непустой шаг (insert W, search W, prefix W или delete W)",
    );
  }
  return commands;
}

/** Узел зеркала Trie в dry-run: дети по рунам и отметка конца слова. */
interface TrieSimNode {
  children: Map<string, TrieSimNode>;
  isEnd: boolean;
}

/**
 * Публичная симуляция сценария (dry-run): холостой прогон команд над
 * зеркалом Trie с пустым корнем и подсчёт живых rune-узлов (корень не
 * считается) — та же семантика, что у серверной checkTrieNodeLimit
 * (visual/server/trie.go) до SSE. Лимит применяется к пиковому размеру
 * Trie — максимуму по состоянию после каждой команды, а не только к
 * финальному. Вставка создаёт узел на каждую отсутствующую руну, удаление
 * снимает отметку конца слова и срезает снизу вверх узлы, которые больше
 * не ведут ни к одному концу слова; промахи search/prefix/delete и
 * повторные вставки топологию не меняют. Превышение лимита — Error до
 * отправки Run. Возвращает пиковое число живых rune-узлов (корень не
 * считается) — публичный итог dry-run: ноль узлов, например, у поиска по
 * пустому Trie, чьи события проверки рёбер ничего не создают.
 */
export function checkTrieNodeLimit(commands: readonly TrieCommand[]): number {
  const root: TrieSimNode = { children: new Map(), isEnd: false };
  let live = 0;
  let peak = 0;
  for (let i = 0; i < commands.length; i++) {
    const cmd = commands[i];
    const runes = Array.from(cmd.word); // руны, не UTF-16 code units
    if (cmd.op === "insert") {
      let node = root;
      for (const rune of runes) {
        let next = node.children.get(rune);
        if (next === undefined) {
          next = { children: new Map(), isEnd: false };
          node.children.set(rune, next);
          live++;
        }
        node = next;
      }
      node.isEnd = true;
    } else if (cmd.op === "delete") {
      // Путь от корня (parent + руна на переходе) — для прунинга снизу вверх.
      const path: Array<{ parent: TrieSimNode; rune: string }> = [];
      let node = root;
      let exists = true;
      for (const rune of runes) {
        const next = node.children.get(rune);
        if (next === undefined) {
          exists = false; // промах по отсутствующему ребру — мутаций нет
          break;
        }
        path.push({ parent: node, rune });
        node = next;
      }
      if (exists && node.isEnd) {
        node.isEnd = false;
        for (let j = path.length - 1; j >= 0; j--) {
          const { parent, rune } = path[j];
          const cur = parent.children.get(rune);
          if (cur === undefined) break; // недостижимо: путь собран по детям
          if (cur.isEnd || cur.children.size > 0) break; // первый живой узел
          parent.children.delete(rune);
          live--;
        }
      }
    }
    // search и prefix топологию не меняют — счётчик не трогают.
    if (live > peak) peak = live;
    if (live > MAX_TRIE_NODES) {
      throw new Error(
        `сценарий превышает лимит живых rune-узлов: разрешено не более ${MAX_TRIE_NODES}, а команда №${i + 1} (${cmd.op} ${cmd.word}) даёт ${live} узлов; сократите сценарий`,
      );
    }
  }
  return peak;
}
