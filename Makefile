build:
	go build -o http cmd/http/*.go

clean:
	if [ -f http ] ; then rm http ; fi
