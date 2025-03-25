package golog

import (
	"fmt"
	"hash/crc32"
)

type color int

const (
	colorBlack color = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite
)

var colorList = [...]color{colorRed, colorGreen, colorYellow, colorBlue, colorMagenta}

func showColor(color color, msg string) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", int(color), msg)
}

func showColorBold(color color, msg string) string {
	return fmt.Sprintf("\033[%d;1m%s\033[0m", int(color), msg)
}

func getColorServiceName(serviceName string) string {
	c := crc32.ChecksumIEEE([]byte(serviceName))
	usedColor := colorList[int(c)%len(colorList)]
	return showColorBold(usedColor, serviceName)
}
