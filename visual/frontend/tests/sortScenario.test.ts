// Тест шва фронта (docs/SPEC.md, «Четвёртый шов»): парсинг массива
// сортировки и сборка URL — ошибки ввода ловятся до Run. Зеркало серверной
// валидации (visual/server/sort.go): грамматика целых чисел (strconv.Atoi)
// и лимит длины визуального Run.
import { describe, expect, it } from "vitest";
import { MAX_SORT_VALUES, parseSortValues } from "../src/scenario";
import { buildSortRunURL } from "../src/sse";

describe("parseSortValues — удачный разбор", () => {
  it("разбирает массив через запятую", () => {
    expect(parseSortValues("3,1,2")).toEqual([3, 1, 2]);
  });

  it("допускает отрицательные числа и пробелы вокруг значений", () => {
    expect(parseSortValues("-1, 2 , -3")).toEqual([-1, 2, -3]);
    expect(parseSortValues("+5,-2")).toEqual([5, -2]);
  });

  it("пустой ввод — допустимый пустой массив", () => {
    expect(parseSortValues("")).toEqual([]);
    expect(parseSortValues("   ")).toEqual([]);
  });

  it("одно значение — массив из одного элемента", () => {
    expect(parseSortValues("42")).toEqual([42]);
  });

  it("лимит: ровно MAX_SORT_VALUES элементов допустимо", () => {
    const values = Array.from({ length: MAX_SORT_VALUES }, (_, i) => i);
    expect(parseSortValues(values.join(","))).toHaveLength(MAX_SORT_VALUES);
  });
});

describe("parseSortValues — ошибки до Run", () => {
  it("отвергает мусор вместо целого числа", () => {
    expect(() => parseSortValues("1,x,3")).toThrow(/не удалось разобрать «x»/);
    expect(() => parseSortValues("пять")).toThrow(/не удалось разобрать «пять»/);
  });

  it("отвергает пустую компоненту между запятыми и запятую в конце", () => {
    expect(() => parseSortValues("1,,3")).toThrow(/не удалось разобрать/);
    expect(() => parseSortValues("1,3,")).toThrow(/не удалось разобрать/);
  });

  it("отвергает дробные и экспоненциальные формы — сервер тоже (strconv.Atoi)", () => {
    expect(() => parseSortValues("1.5,2")).toThrow(/не удалось разобрать «1.5»/);
    expect(() => parseSortValues("1e2")).toThrow(/не удалось разобрать/);
  });

  it("отвергает превышение лимита длины", () => {
    const values = Array.from({ length: MAX_SORT_VALUES + 1 }, (_, i) => i);
    expect(() => parseSortValues(values.join(","))).toThrow(/максимум 100/);
  });

  it("отвергает значения за пределами безопасного целого", () => {
    expect(() => parseSortValues("9007199254740992")).toThrow(/диапазона JavaScript/);
  });
});

// Ожидаемый маршрут — литералом в таблице, а не интерполяцией входа
// билдера: тест сверяет URL с независимым ожиданием.
describe.each([
  { algorithm: "bubble-sort", route: "/api/run/bubble-sort" },
  { algorithm: "selection-sort", route: "/api/run/selection-sort" },
  { algorithm: "insertion-sort", route: "/api/run/insertion-sort" },
  { algorithm: "merge-sort", route: "/api/run/merge-sort" },
  { algorithm: "quick-sort", route: "/api/run/quick-sort" },
] as const)("buildSortRunURL — $route", ({ algorithm, route }) => {
  it("собирает URL маршрута сортировки с массивом значений", () => {
    expect(buildSortRunURL(algorithm, [3, 1, 2])).toBe(`${route}?values=3%2C1%2C2`);
  });

  it("пустой массив едет как есть — сервер авторитетен", () => {
    expect(buildSortRunURL(algorithm, [])).toBe(`${route}?values=`);
  });
});
