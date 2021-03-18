# go-waku

### Examples

1. Start a node that will continously send a "Hey" message to the default waku topic
```
go run waku.go --port 33331 --hey --nodekey 112233445566778899001122334455667788990011223344556677889900AAAA
```


2. Node that will connect to the "Hey" node and store messages
```
go run waku.go --port 44441 --store --start-store --nodekey 112233445566778899001122334455667788990011223344556677889900BBBB --staticnodes /ip4/127.0.0.1/tcp/33331/p2p/16Uiu2HAmK4LBcnRWJctkhkbSFYBbNApYYbSeo8fkC496weKU1R5B
```


3. Node that will connect to the Store node and use it to retrieve messages retrieve messages
```
go run waku.go --port 0 --store --nodekey 112233445566778899001122334455667788990011223344556677889900CCCC --staticnodes /ip4/127.0.0.1/tcp/44441/p2p/16Uiu2HAmVGaeLcyYoGjEKe8uPg5WVMVXyLDegbfyABKn95tH7jsU --storenode /ip4/127.0.0.1/tcp/44441/p2p/16Uiu2HAmVGaeLcyYoGjEKe8uPg5WVMVXyLDegbfyABKn95tH7jsU --query

```

4. Node that will listen to the default topic by connecting to a node
```
go run waku.go --port 0 --store --nodekey 112233445566778899001122334455667788990011223344556677889900DDDD --staticnodes /ip4/127.0.0.1/tcp/44441/p2p/16Uiu2HAmVGaeLcyYoGjEKe8uPg5WVMVXyLDegbfyABKn95tH7jsU --listen
```