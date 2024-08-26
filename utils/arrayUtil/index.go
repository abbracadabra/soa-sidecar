package arrayUtil

import (
	"math/rand"
	"time"
)

func MapSlice[T any, R any](input []T, transform func(T) R) []R {
	output := make([]R, len(input))
	for i, v := range input {
		output[i] = transform(v)
	}
	return output
}

func Shuffle[T any](slice []T) {
	rand.Seed(time.Now().UnixNano()) // 使用当前时间为种子初始化随机数生成器
	n := len(slice)
	for i := n - 1; i > 0; i-- {
		j := rand.Intn(i + 1)                   // 生成0到i之间的随机整数
		slice[i], slice[j] = slice[j], slice[i] // 交换元素
	}
}

func DeepCopy[T any](src []T) []T {
	dst := make([]T, len(src))
	copy(dst, src)
	return dst
}
