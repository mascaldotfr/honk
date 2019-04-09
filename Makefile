
all: honk

honk: *.go
	go build -o honk

clean:
	rm -f honk
