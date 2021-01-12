# Mysql HTTP plugin
A mysql plugin for sending message to an HTTP endpoint.

This plugin was used in production when async messages became a need for the infrastructure. 
Meanwhile, we got a NSQ cluster deployed and this is plugin is not used and/or maintained.

I'm sharing this with you because I believe some of you may find it: a good starting point in building your own MYSQL plugins.

1. Install prerequisites
```
apt-get install libmysqlclient-dev
```
2. Build plugin
```
make build 
```

3. Upload the plugin into server by ftp and copy the file in the mysql plugin path
```mysql
mysql> show variables like 'plugin_dir';
+---------------+--------------------------+
| Variable_name | Value                    |
+---------------+--------------------------+
| plugin_dir    | /usr/lib64/mysql/plugin/ |
+---------------+--------------------------+
1 row in set (0.00 sec)
```

4. Install the plugin
```mysql
INSTALL PLUGIN http_notify SONAME 'http_notify_udf.so';
CREATE FUNCTION http_notify RETURNS STRING SONAME 'http_notify_udf.so';
```

5. Configure endpoint 
```ini
[my.cnf]
http_notify_endpoint=http://webhook.devs.local:8000/hooks/v1
http_notify_username=
http_notify_password=
```

Notes:
* Endpoint is the base URL for your api
* Credentials are optional and they are enabling Basic Authentication with the endpoint


## Use the plugin
### POST
```sql
SELECT http_notify("POST","users",'{"id":1,"username":"test","email":"test@domain.com"}')
```
will queue an HTTP 
* method: POST
* url:  `http://webhook.devs.local:8000/hooks/v1/user` 
* payload `{"id":1,"username":"test","email":"test@domain.com"}`

### PUT
```sql
SELECT http_notify("PUT","users",'{"id":1,"username":"test","email":"test@domain.com"}')
```
will queue an HTTP
* method: PUT
* url:  `http://webhook.devs.local:8000/hooks/v1/user/1`
* payload `{"id":1,"username":"test","email":"test@domain.com"}`

### DELETE
```sql
SELECT http_notify("DELETE","users",'{"id":1,"username":"test","email":"test@domain.com"}')
```
will queue an HTTP
* method: DELETE
* url:  `http://webhook.devs.local:8000/hooks/v1/user/1`
* payload `{"id":1}`

Note:
* In order to get this working properly there is a contrain that your payload needs to be a JSON payload containing the "id" field.

