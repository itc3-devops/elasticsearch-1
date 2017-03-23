import sys, os, subprocess, json, shutil, yaml
from elasticsearch import Elasticsearch

Flag = {}

secret_data_dir = "/var/credentials"


def set_osm_config():
    with open(secret_data_dir+"/provider") as f:
        lines = f.readlines()
        if len(lines) == 0:
            print "Provider missing"
            exit(1)
        provider = lines[0].rstrip('\n')

    with open(secret_data_dir + "/config") as f:
        config = json.load(f)

    context = dict(
        config=config,
        name="cloud",
        provider=provider,
    )

    data = {
        "contexts": [context],
        "current-context": "cloud",
    }

    home = os.environ['HOME']
    osm_config_path = home + '/.osm'
    if not os.path.exists(osm_config_path):
        os.makedirs(osm_config_path)
    with open(osm_config_path+"/config", 'w') as outfile:
        yaml.safe_dump(data, outfile, default_flow_style=False)


def backup_process():
    print "Backup process starting..."

    es = Elasticsearch(hosts=[{'host': Flag["host"], 'port': 9200}], timeout=120)
    indices = es.indices.get_alias()

    print "Total indices: " + str(len(indices))
    path = '/var/dump-backup/'+Flag["snapshot"]
    shutil.rmtree(path, ignore_errors=True)

    if not os.path.exists(path):
        os.makedirs(path)

    for index in indices:
        code = subprocess.call(['./utils.sh', "backup", Flag["host"], Flag["snapshot"], index])
        if code != 0:
            print "Fail to take backup for index: "+index
            exit(1)

    file_pointer = open(path+"/indices.txt", "wb")
    for index in indices:
        print>>file_pointer, index
    file_pointer.close()

    code = subprocess.call(['./utils.sh', "push", Flag["bucket"], Flag["folder"], Flag["snapshot"]])
    if code != 0:
        print "Fail to push backup files to cloud..."
        exit(1)


def restore_process():
    print "Restore process starting..."

    code = subprocess.call(['./utils.sh', "pull", Flag["bucket"], Flag["folder"], Flag["snapshot"]])
    if code != 0:
        print "Fail to pull backup files from cloud..."
        exit(1)

    path = '/var/dump-restore/'+Flag["snapshot"]
    file_pointer = open(path+"/indices.txt", "r")
    for index in file_pointer.readlines():
        index = index.rstrip("\n")
        code = subprocess.call(['./utils.sh', "restore", Flag["host"], Flag["snapshot"], index])
        if code != 0:
            print "Fail to restore index: "+index
            exit(1)


def main(argv):
    for flag in argv:
        if flag[:2] != "--":
            continue
        v = flag.split("=", 1)
        Flag[v[0][2:]]=v[1]

    for flag in ["process", "host", "bucket", "folder", "snapshot"]:
        if flag not in Flag:
            print '--%s is required'%flag
            exit(1)
            return

    set_osm_config()

    if Flag["process"] == "backup":
        backup_process()
    elif Flag["process"] == "restore":
        restore_process()

    exit(0)

if __name__ == "__main__":
    main(sys.argv[1:])