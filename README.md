# ntraffstat
a nginx traffic statistics/analyse tool

it can

1. log every [5 minutes/1 huor/1 day] 's [ip / host / url / file on disk] requests and body size
2. realtime statistics
3. history log saved on disk
4. log url referers relationship

it helps you

1. find which ip or which file cause huge traffic
2. compare files / hosts and their traffic usages, help you optimize load speed

how to use

1. build with golang ```go build```
2. edit allow_ip and allow_users file
3. config nginx, add log format (only support this format)

	```log_format  traffic  '$remote_addr "$host$request_uri" $body_bytes_sent "$request_filename" "$http_referer"';```

4. set nginx access_log to ```/tmp/nginx_traffic.log``` which is a fifo created by ntraffstat

	```access_log  /tmp/nginx_traffic.log traffic;```

5. start ntraffstat and then reload nginx
