// Пакет algorithms — общий контракт библиотеки: протокол событий шагов (ADR-0001).
//
// Каждый алгоритм определяет свои типы событий; контракт приёма — общий.
// Алгоритмы работают в тихом режиме (quiet mode) без эмиттера и опционально
// стреляют событиями в Emitter для визуализации.
package algorithms

// Emitter — приёмник событий шагов алгоритма.
type Emitter[E any] interface {
	Emit(event E)
}

// EmitterFunc — адаптер «функция → эмиттер», для простых подписчиков.
type EmitterFunc[E any] func(event E)

// Emit удовлетворяет Emitter[E].
func (f EmitterFunc[E]) Emit(event E) { f(event) }
