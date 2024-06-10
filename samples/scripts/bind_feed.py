import sys
import fcntl
import json
import requests

if len(sys.argv) != 2:
    raise RuntimeError("Usage: python script.py <AnotherFeedId>")
AnotherFeedId = sys.argv[1]
if AnotherFeedId == "" :
    raise RuntimeError("Usage: python script.py <AnotherFeedId>")

lock_fd = open(__file__, 'r')
fcntl.flock(lock_fd, fcntl.LOCK_EX)

input_data = sys.stdin.read()

json_data = json.loads(input_data)
id_value = json_data.get('id')

print(id_value)

url = f'http://localhost/api/feed/{AnotherFeedId}'
headers = {'Content-Type': 'application/json'}
payload = {'id': id_value}

response = requests.post(url, headers=headers, json=payload)

print(response.status_code)
print(response.text)
