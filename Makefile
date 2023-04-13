watch_demo:
	find cmd/demo | grep 'html\|go'  | entr -r -c go run cmd/demo/demo.go

watch_server:
	find cmd/server | entr -c -r bash -c 'cd cmd/server && flask run -p 8081'

watch_webapp:
	find cmd/webapp | grep 'html\|go\|js' | entr -c -r go run cmd/webapp/webapp.go