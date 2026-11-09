// События BST-операций (ADR-0001): каждое событие — атомарный шаг
// вставки, поиска или удаления. Путь узла от корня (строка из L/R,
// как в обходах; NodeID) — идентификатор узла, не зависящий от значений.
//
// По событиям восстанавливаются: сравнения и переходы по ветвям
// (BSTCompare — с выбранной ветвью), исход операции (BSTInserted,
// BSTDuplicate, BSTFound, BSTNotFound, BSTRemoved) и структурные
// изменения, включая поиск преемника (BSTMinStep, BSTSuccessor) и его
// удаление (BSTRemoved на пути преемника). Один путь задаётся одной
// координатой дерева — события узнаваемы и без снимка дерева.
package trees

import "github.com/EvgeniiIvanov/algo/algorithms"

// BSTEvent — маркер-интерфейс событий BST (sealed): вне пакета свои
// события реализовать нельзя.
type BSTEvent interface{ isBSTEvent() }

// BSTCompare — сравнение искомого ключа операции с узлом на пути Path:
// Key меньше Value — ветвь "L" (влево), больше — "R" (вправо), равен —
// "" (перехода нет: ключ в этом узле).
type BSTCompare struct {
	Path   string // путь узла сравнения от корня ("": корень, "L", "LR", ...)
	Key    int    // искомый ключ текущей операции
	Value  int    // значение узла, с которым сравнили
	Branch string // выбранная ветвь после сравнения: "L", "R"; "" — равенство
}

// BSTInserted — новый узел с ключом создан в пустом месте Path:
// структурное изменение, дерево выросло на один узел.
type BSTInserted struct {
	Path string // путь нового узла от корня
	Key  int    // вставленный ключ
}

// BSTDuplicate — ключ уже есть в узле на пути Path: повторная вставка
// не меняет дерево, новый узел не добавлен.
type BSTDuplicate struct {
	Path string // путь существующего узла с этим ключом
	Key  int    // повторно вставляемый ключ
}

// BSTFound — ключ найден в узле на пути Path.
type BSTFound struct {
	Path string // путь найденного узла
	Key  int    // найденный ключ
}

// BSTNotFound — спуск дошёл до пустого места Path, ключа в дереве нет:
// промах — штатный исход поиска и удаления (не ошибка).
type BSTNotFound struct {
	Path string // место, где ключ был бы ("" — пустое дерево)
	Key  int    // ненайденный ключ
}

// BSTRemoved — узел с ключом удалён с пути Path: на его место встал
// единственный потомок или пустота (лист, один потомок, узел-преемник).
// У узла с двумя потомками сам узел не удаляется — см. BSTSuccessor.
type BSTRemoved struct {
	Path string // путь удалённого узла
	Key  int    // ключ удалённого узла
}

// BSTMinStep — поиск минимума правого поддерева (inorder-successor)
// стоит на узле Path со значением Value; спуск идёт влево, пока есть
// левый потомок. Последний BSTMinStep цепочки — выбранный преемник.
type BSTMinStep struct {
	Path  string // путь посещённого узла
	Value int    // значение посещённого узла
}

// BSTSuccessor — у удаляемого узла Path два потомка: выбран
// inorder-successor — узел SuccessorPath со значением Value (минимум
// правого поддерева). Значение преемника переходит удаляемому узлу,
// сам узел-преемник затем удаляется (BSTRemoved на SuccessorPath).
type BSTSuccessor struct {
	Path          string // путь удаляемого узла с двумя потомками
	SuccessorPath string // путь преемника — минимума правого поддерева
	Value         int    // значение преемника (новое значение узла Path)
}

func (BSTCompare) isBSTEvent()   {}
func (BSTInserted) isBSTEvent()  {}
func (BSTDuplicate) isBSTEvent() {}
func (BSTFound) isBSTEvent()     {}
func (BSTNotFound) isBSTEvent()  {}
func (BSTRemoved) isBSTEvent()   {}
func (BSTMinStep) isBSTEvent()   {}
func (BSTSuccessor) isBSTEvent() {}

// InsertWithEmitter — Insert в режиме наблюдения: каждое сравнение
// (BSTCompare) и структурное изменение (BSTInserted, BSTDuplicate)
// уходят в эмиттер. Поведение и результат — те же, что у тихого Insert.
func InsertWithEmitter(root *Node, key int, em algorithms.Emitter[BSTEvent]) (*Node, bool) {
	return insert(root, key, "", em)
}

// SearchWithEmitter — Search в режиме наблюдения: сравнения (BSTCompare)
// и исход (BSTFound, BSTNotFound). Результат — тот же, что у тихого Search.
func SearchWithEmitter(root *Node, key int, em algorithms.Emitter[BSTEvent]) (*Node, bool) {
	return search(root, key, em)
}

// DeleteWithEmitter — Delete в режиме наблюдения: сравнения (BSTCompare),
// поиск преемника (BSTMinStep, BSTSuccessor) и структурные изменения
// (BSTRemoved). Результат — тот же, что у тихого Delete.
func DeleteWithEmitter(root *Node, key int, em algorithms.Emitter[BSTEvent]) (*Node, bool) {
	return remove(root, key, "", em)
}

// emitBST вызывает эмиттер, если он передан (nil-check — деталь реализации).
func emitBST(em algorithms.Emitter[BSTEvent], event BSTEvent) {
	if em == nil {
		return
	}
	em.Emit(event)
}
