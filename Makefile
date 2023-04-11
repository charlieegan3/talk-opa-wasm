watch_server:
	find cmd/server | entr -c -r bash -c 'cd cmd/server && flask run -p 8081'