get:
	go get github.com/diamondburned/gotk4-layer-shell/pkg/gtklayershell
	go get github.com/diamondburned/gotk4/pkg
	go get github.com/joshuarubin/go-sway
	go get github.com/allan-simon/go-singleinstance
	go get "github.com/sirupsen/logrus"

build:
	go build -v -o bin/dock-mango .

install:
	-pkill -f dock-mango
	sleep 1
	mkdir -p /usr/share/dock-mango
	cp -r images /usr/share/dock-mango
	cp config/* /usr/share/dock-mango
	cp bin/dock-mango /usr/bin

uninstall:
	rm -r /usr/share/dock-mango
	rm /usr/bin/dock-mango

run:
	go run .
