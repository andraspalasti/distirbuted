MAELSTROM_URL=https://github.com/jepsen-io/maelstrom/releases/download/v0.2.3/maelstrom.tar.bz2

all: build

clean:
	rm -rf maelstrom/
	rm -rf store/

# Download maelstrom and extract it than delete the tar
maelstrom:
	wget -c $(MAELSTROM_URL) -q -O - | tar -xz
	chmod +x ./maelstrom/maelstrom

build:
	go build -o ./maelstrom/echo ./echo/
	go build -o ./maelstrom/uniqueids ./uniqueids/
	go build -o ./maelstrom/broadcast ./broadcast/
	go build -o ./maelstrom/counter ./counter/

test-echo: maelstrom build
	cd maelstrom/; \
	./maelstrom test -w echo --bin ./echo --node-count 1 --time-limit 10

test-uniqueids: maelstrom build
	cd maelstrom/; \
	./maelstrom test -w unique-ids --bin ./uniqueids --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition

test-broadcast: maelstrom build
# For part a: ./maelstrom test -w broadcast --bin ./broadcast --node-count 1 --time-limit 20 --rate 10
# For part b: ./maelstrom test -w broadcast --bin ./broadcast --node-count 5 --time-limit 20 --rate 10
# For part c: ./maelstrom test -w broadcast --bin ./broadcast --node-count 5 --time-limit 20 --rate 10 --nemesis partition
# For part d and e the command is the same
	cd maelstrom/; \
	./maelstrom test -w broadcast --bin ./broadcast --node-count 25 --time-limit 20 --rate 100 --latency 100

test-counter: maelstrom build
	cd maelstrom/; \
	./maelstrom test -w g-counter --bin ./counter --node-count 3 --rate 100 --time-limit 20 --nemesis partition
