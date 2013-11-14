#!/bin/sh

set -x

# Cleanup
rm -rf docker-registry

# Setup the environment
export SETTINGS_FLAVOR=test
export DOCKER_REGISTRY_CONFIG=config_test.yml
export PYTHONPATH=$(pwd)/docker-registry/test

# Get latest docker registry
git clone -q https://github.com/dotcloud/docker-registry.git
cd docker-registry
sed -Ei "s#(boto_bucket: ).+#\1_env:S3_BUCKET#" config_test.yml

# Get dependencies
pip install -q -r requirements.txt
pip install -q -r test-requirements.txt
pip install -q tox

# Run registry tests
tox || exit 1
python -m unittest discover -p s3.py -s test || exit 1
python -m unittest discover -p workflow.py -s test; exit_status=$?

rest_url='http://docker-ci-report.dotcloud.com:8080/admin/api/v1/services/registry/events'
if [ $exit_status -eq 0 ]; then
    curl -d message="Registry tests succeeded" -d status=up $rest_url
else
    curl -d message="Registry tests failed" -d status=down $rest_url
fi

exit $exit_status
