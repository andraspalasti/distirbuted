MAELSTROM_URL=https://github.com/jepsen-io/maelstrom/releases/download/v0.2.3/maelstrom.tar.bz2

all: test

clean:
	rm -rf maelstrom/
	rm -rf store/

# Download maelstrom and extract it than delete the tar
maelstrom:
	wget -c $(MAELSTROM_URL) -q -O - | tar -xz
	chmod +x ./maelstrom/maelstrom

build:
	go build -o ./maelstrom/echo ./echo/

test: maelstrom build
	./maelstrom/maelstrom test -w echo --bin ./maelstrom/echo --node-count 1 --time-limit 10
