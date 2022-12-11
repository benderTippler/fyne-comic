go build -ldflags="-H windowsgui"

fyne package -os windows

// 推荐 https://github.com/fyne-io/fyne-cross
fyne-cross windows -arch=*
