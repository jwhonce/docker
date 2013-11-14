'''Submitting docker-ci index status sample'''

import requests, sys

docker_status_server = 'docker-ci-report.dotcloud.com'

# Base url for stashboard rest requests
base_url = "http://{}:8080/admin/api/v1".format(docker_status_server)

# Create a new event with the given status and given service
data = { "message": "Index tests succeeded",
         "status": 'up' }

# Submit status
response = requests.post(base_url + "/services/registry/events", data)

# Signal issue if necessary
if response.status_code != 200:
    sys.exit(1)
