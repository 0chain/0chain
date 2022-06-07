# blobber/

### Module: blobbercore

```sh
File: blobber/code/go/0chain.net/blobbercore/handler/handler.go
```
> object operations

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/file/upload/{allocation} | UploadHandler |
| /v1/file/download/{allocation} | DownloadHandler |
| /v1/file/rename/{allocation} | RenameHandler |
| /v1/file/copy/{allocation} | CopyHandler |
| /v1/file/attributes/{allocation} | UpdateAttributesHandler |
| /v1/connection/commit/{allocation} | CommitHandler |
| /v1/file/commit-meta-txn/{allocation} | CommitMetaTxnHandler |
| /v1/file/collaborator/{allocation} | CollaboratorHandler |
| /v1/file/calculatehash/{allocation} | CalculateHashHandler |

> object info related apis

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /allocation | AllocationHandlerr |
| /v1/file/download/{allocation} | FileMetaHandler |
| /v1/file/rename/{allocation} | FileStatsHandler |
| /v1/file/copy/{allocation} | ListHandler |
| /v1/file/attributes/{allocation} | ObjectPathHandler |
| /v1/connection/commit/{allocation} | ReferencePathHandler |
| /v1/file/commit-meta-txn/{allocation} | ObjectTreeHandler |


> admin related

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /_debug | DumpGoRoutines |
| /_config | GetConfig |
| /_stats | stats.StatsHandler |
| /_statsJSON | stats.StatsJSONHandler |
| /_cleanupdisk | CleanupDiskHandler |
| /getstats | stats.GetStatsHandler |


### Module: validatorcore

```sh
File: blobber/code/go/0chain.net/validatorcore/storage/handler.go
```

> SetupHandlers sets up the necessary API end points

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/storage/challenge/new | ChallengeHandler |
| /debug | DumpGoRoutines |







