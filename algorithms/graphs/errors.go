package graphs

import "errors"

// Ошибки публичного API пакета graphs; сравнение — через errors.Is.
var (
	// ErrNegativeNumberOfVertices — число вершин отрицательное
	// (ноль валиден: это пустой граф).
	ErrNegativeNumberOfVertices = errors.New("number of vertices must not be negative")
	// ErrVertexNotFound — вершина вне диапазона 0..n-1.
	ErrVertexNotFound = errors.New("vertex not found")
	// ErrInvalidAdjacencyMatrix — матрица смежности не квадратная.
	ErrInvalidAdjacencyMatrix = errors.New("adjacency matrix must be square")
	// ErrDuplicateEdge — ребро уже существует (с учётом режима графа).
	ErrDuplicateEdge = errors.New("duplicate edge")
)
