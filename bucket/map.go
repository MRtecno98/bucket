package bucket

import "cmp"

type BiMap[K1 cmp.Ordered, K2 cmp.Ordered, V any] struct {
	first  map[K1]V
	second map[K2]V

	keyfunc func(el V) (first K1, second K2)
}

func NewBiMap[K1 cmp.Ordered, K2 cmp.Ordered, V any](
	keyfunc func(el V) (first K1, second K2)) *BiMap[K1, K2, V] {
	return &BiMap[K1, K2, V]{
		first:   make(map[K1]V),
		second:  make(map[K2]V),
		keyfunc: keyfunc,
	}
}

func (bm *BiMap[K1, K2, V]) Put(value V) {
	first, second := bm.keyfunc(value)
	bm.first[first] = value
	bm.second[second] = value
}

func (bm *BiMap[K1, K2, V]) GetFirst(key K1) (V, bool) {
	value, ok := bm.first[key]
	return value, ok
}

func (bm *BiMap[K1, K2, V]) GetSecond(key K2) (V, bool) {
	value, ok := bm.second[key]
	return value, ok
}

func (bm *BiMap[K1, K2, V]) DeleteFirst(key K1) {
	value, ok := bm.first[key]
	if ok {
		delete(bm.first, key)
		_, sk := bm.keyfunc(value)
		delete(bm.second, sk)
	}
}

func (bm *BiMap[K1, K2, V]) DeleteSecond(key K2) {
	value, ok := bm.second[key]
	if ok {
		delete(bm.second, key)
		fk, _ := bm.keyfunc(value)
		delete(bm.first, fk)
	}
}

func (bm *BiMap[K1, K2, V]) Values() []V {
	var values []V
	for _, v := range bm.first {
		values = append(values, v)
	}
	return values
}

type SymmetricBiMap[K cmp.Ordered, V any] struct {
	BiMap[K, K, V]
}

func NewSymmetricBiMap[K cmp.Ordered, V any](
	keyfunc func(el V) (first K, second K)) *SymmetricBiMap[K, V] {
	return &SymmetricBiMap[K, V]{
		BiMap: BiMap[K, K, V]{
			first:   make(map[K]V),
			second:  make(map[K]V),
			keyfunc: keyfunc,
		},
	}
}

func (sbm *SymmetricBiMap[K, V]) GetAny(key K) (V, bool) {
	value, ok := sbm.GetFirst(key)
	if ok {
		return value, ok
	}

	return sbm.GetSecond(key)
}

func (sbm *SymmetricBiMap[K, V]) Delete(key K) {
	sbm.DeleteFirst(key)
	sbm.DeleteSecond(key)
}

func (sbm *SymmetricBiMap[K, V]) GetStrict(key K) (V, bool) {
	v, a := sbm.GetFirst(key)
	_, b := sbm.GetSecond(key)

	return v, a && b
}
