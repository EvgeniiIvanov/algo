// Тест шва фронта: публичный контракт общего модуля decimalInteger —
// единственного места с грамматикой Go strconv.Atoi во фронтенде.
// Проверяем различение syntax / unsafe-range (введение ревью тикета 13)
// и точную передачу обрезанной строки в input — caller'ы строят на ней
// свои контекстные сообщения. Поведение конкретных парсеров (стек,
// бинарный поиск) покрыто их швовыми тестами через этот контракт.
import { describe, expect, it } from "vitest";
import { parseDecimalInteger } from "../src/decimalInteger";

describe("parseDecimalInteger", () => {
  it("разбирает десятичное целое: знак, ведущие нули, trim по краям", () => {
    expect(parseDecimalInteger("7")).toEqual({ ok: true, value: 7 });
    expect(parseDecimalInteger(" +2 ")).toEqual({ ok: true, value: 2 });
    expect(parseDecimalInteger("007")).toEqual({ ok: true, value: 7 });
  });

  it("принимает границу безопасного целого в обе стороны", () => {
    expect(parseDecimalInteger("9007199254740991")).toEqual({ ok: true, value: 9007199254740991 });
    expect(parseDecimalInteger("-9007199254740991")).toEqual({
      ok: true,
      value: -9007199254740991,
    });
  });

  it("сообщает syntax для пустой строки, мусора, дробей, экспоненты и hex", () => {
    const bad: string[] = ["", "   ", "три", "2.5", "2.0", "1e2", "0x10", "1 2", "-", "+"];
    for (const raw of bad) {
      const result = parseDecimalInteger(raw);
      if (!result.ok) {
        expect(result.reason).toBe("syntax");
        expect(result.input).toBe(raw.trim());
      } else {
        throw new Error(`ожидали syntax, получили value=${result.value} для «${raw}»`);
      }
    }
  });

  it("сообщает unsafe-range за пределами безопасного целого: не округляет и не путает с syntax", () => {
    for (const raw of ["9007199254740992", "-9007199254740992", "99999999999999999999"]) {
      const result = parseDecimalInteger(raw);
      if (!result.ok) {
        expect(result.reason).toBe("unsafe-range");
        // Формально грамматически верно: ошибка именно про диапазон.
        expect(result.input).toBe(raw.trim());
      } else {
        throw new Error(`ожидали unsafe-range, получили value=${result.value} для «${raw}»`);
      }
    }
  });
});
