package utils

func toSnake(str string) string {
	runes := []rune(str)
	var p int
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if c >= 'A' && c <= 'Z' {
			runes[i] = c - ('A' - 'a')
			if p+1 < i {
				tmp := append([]rune{'_'}, runes[i:]...)
				runes = append(runes[0:i], tmp...)
				i++
			}
			p = i
		}
	}
	return string(runes)
}
