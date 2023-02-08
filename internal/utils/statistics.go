package utils

import (
	"math"
	"sort"
)

func FindMinAndMax(s []int) (min int, max int) {
	if len(s) == 0 {
		return 0, 0
	}
	min = s[0]
	max = s[0]
	for _, value := range s {
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}
	return min, max
}

func Avg(s []int) (avg int) {
	count := len(s)
	if count == 0 {
		return 0
	}
	total := 0
	for _, v := range s {
		total += v
	}
	avg = total / count
	return avg
}

func NinetyFifth(s []int) int {
	count := len(s)
	if count == 0 {
		return 0
	}
	sort.Ints(s)
	indexNinetyFifth := int(math.Ceil(float64(count)*0.95)) - 1
	return s[indexNinetyFifth]
}
