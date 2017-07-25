
# CoreOS Update Logger

## Description
Simple app that sends CoreOS update metrics to ElasticSearch so Kibana can be used as a dashboard to track your CoreOS updates

### Build
```
go get
go build
```

Linux
```
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .
```

### Docker Build


Docker
```
docker build -t coreos-update-logger .

```

or 

The Dockerfile.build in this project uses a multi stage docker build which requires docker >= 17.05 

```
docker build -t coreos-update-logger -f Dockerfile.build .
```

### Example  
CLI  
```
coreos-update-logger --url https://<elasticsearch_endpoint>:9200 --host <coreos_host_identifier> --lock_smith /run/systemd/system/locksmithd.service.d/20-cloudinit.conf --os_rel /etc/os-release --update_conf /etc/coreos/update.conf --uptime /proc/uptime
```

Docker
```
docker run -ti --net=weave -v /run/systemd/system/locksmithd.service.d/20-cloudinit.conf:/20-cloudinit.conf:ro -v /etc/os-release:/os-release:ro -v /etc/coreos/update.conf:/update.conf:ro -v /proc/uptime:/uptime:ro -e HOST=${COREOS_PRIVATE_IPV4} -e URL=http://<elasticsearch_endpoint>:9200 -e FREQ=2 -e ENV=dev thefoo/coreos-update-logger:latest

docker run -d --name coreos-update-logger --net=host -v /run/systemd/system/locksmithd.service.d/20-cloudinit.conf:/20-cloudinit.conf:ro -v /etc/os-release:/os-release:ro -v /etc/coreos/update.conf:/update.conf:ro -v /proc/uptime:/uptime:ro -e HOST=${COREOS_PRIVATE_IPV4} -e URL=http://<elasticsearch_endpoint>:9200 -e FREQ=600 -e ENV=dev thefoo/coreos-update-logger:latest

```

## Documentation

Update information is gather by these files
/etc/coreos/update.conf
/etc/os-release
/proc/uptime
Drop-In: /run/systemd/system/locksmithd.service.d
         └─20-cloudinit.conf

### options
Available command line optoins  

```
   --indexname value, -i value      ElasticSearch indexName (default: "coreupdate") [$INDEX_NAME]
   --url value, -u value            ElasticSearch endpoint [$URL]
   --freq value, -f value           Frequency in seconds on when data is sent to ElasticSearch (default: 600) [$FREQ]
   --host value                     CoreOS Hostname or Ip [$HOST]
   --env value, -e value            Environment tag (optional) [$ENV]
   --lock_smith value, -l value     Location of locksmithd config /run/systemd/system/locksmithd.service.d/ (optional) (default: "20-cloudinit.conf") [$LOCK_SMITH]
   --os_rel value, -o value         Location of /etc/os-release file (optional)  (default: "os-release") [$OS_REL]
   --update_conf value, --uc value  Location of /etc/coreos/update.conf file (optional)  (default: "update.conf") [$UPDATE_CONF]
   --uptime value, --up value       Location of /proc/uptime file (optional)  (default: "uptime") [$UPTIME]
   --help, -h                       show help
   --version, -v                    print the version
```
