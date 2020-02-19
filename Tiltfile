watch_file('helm/environments/dev/secrets.yaml')
watch_file('helm/environments/dev/config.yaml')
k8s_yaml(local('helm secrets template quadlek-dev ./helm/quadlek -f helm/environments/dev/secrets.yaml -f helm/environments/dev/config.yaml | head -n -1'))

docker_build("quadlek", ".")