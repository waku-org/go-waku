module github.com/waku-org/go-waku

go 1.20

replace github.com/ethereum/go-ethereum v1.10.26 => github.com/status-im/go-ethereum v1.10.25-status.4

require (
	github.com/btcsuite/btcd/btcec/v2 v2.2.1
	github.com/cruxic/go-hmac-drbg v0.0.0-20170206035330-84c46983886d
	github.com/ethereum/go-ethereum v1.10.26
	github.com/golang-migrate/migrate/v4 v4.15.2
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/gorilla/rpc v1.2.0
	github.com/ipfs/go-ds-sql v0.3.0
	github.com/ipfs/go-log/v2 v2.5.1
	github.com/libp2p/go-libp2p v0.29.2
	github.com/libp2p/go-libp2p-pubsub v0.9.3
	github.com/libp2p/go-msgio v0.3.0
	github.com/mattn/go-sqlite3 v1.14.17
	github.com/multiformats/go-multiaddr v0.10.1
	github.com/stretchr/testify v1.8.4
	github.com/syndtr/goleveldb v1.0.1-0.20220614013038-64ee5596c38a // indirect
	github.com/urfave/cli/v2 v2.24.4
	go.opencensus.io v0.24.0
	go.uber.org/zap v1.24.0
	golang.org/x/sync v0.3.0
)

require (
	github.com/gorilla/mux v1.8.0
	github.com/waku-org/go-discover v0.0.0-20221209174356-61c833f34d98
	golang.org/x/text v0.12.0
)

require (
	github.com/cenkalti/backoff/v4 v4.1.2
	github.com/go-chi/chi/v5 v5.0.0
	github.com/jackc/pgx/v5 v5.4.1
	github.com/waku-org/go-libp2p-rendezvous v0.0.0-20230628220917-7b4e5ae4c0e7
	github.com/waku-org/go-noise v0.0.4
	github.com/waku-org/go-zerokit-rln v0.1.14-0.20231215065255-ab22cb27db30
	github.com/wk8/go-ordered-map v1.0.0
)

require (
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/btcsuite/btcd v0.20.1-beta // indirect
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/pprof v0.0.0-20230705174524-200ffdc848b8 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.2 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.8.0 // indirect
	github.com/jackc/pgerrcode v0.0.0-20201024163028-a0d42d470451 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.0.7 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgtype v1.6.2 // indirect
	github.com/jackc/pgx/v4 v4.10.1 // indirect
	github.com/lib/pq v1.10.4 // indirect
	github.com/onsi/ginkgo/v2 v2.11.0 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/quic-go/qtls-go1-19 v0.3.3 // indirect
	github.com/quic-go/qtls-go1-20 v0.2.3 // indirect
	github.com/quic-go/quic-go v0.36.4 // indirect
	github.com/quic-go/webtransport-go v0.5.3 // indirect
	github.com/rjeczalik/notify v0.9.3 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/status-im/status-go/extkeys v1.1.2 // indirect
	github.com/waku-org/go-zerokit-rln-apple v0.0.0-20230916172309-ee0ee61dde2b // indirect
	github.com/waku-org/go-zerokit-rln-arm v0.0.0-20230916171929-1dd9494ff065 // indirect
	github.com/waku-org/go-zerokit-rln-x86_64 v0.0.0-20230916171518-2a77c3734dd1 // indirect
	go.uber.org/dig v1.17.0 // indirect
	go.uber.org/fx v1.20.0 // indirect
	golang.org/x/xerrors v0.0.0-20220517211312-f3a8303e98df // indirect
)

require (
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/beevik/ntp v0.3.0
	github.com/benbjohnson/clock v1.3.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/containerd/cgroups v1.1.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/davidlazar/go-crypto v0.0.0-20200604182044-b73af7476f6c // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/elastic/gosigar v0.14.2 // indirect
	github.com/flynn/noise v1.0.0 // indirect
	github.com/francoispqt/gojay v1.2.13 // indirect
	github.com/go-ole/go-ole v1.2.1 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/gopacket v1.1.19 // indirect
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d
	github.com/huin/goupnp v1.2.0 // indirect
	github.com/ipfs/go-cid v0.4.1 // indirect
	github.com/ipfs/go-datastore v0.6.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/jbenet/go-temp-err-catcher v0.1.0 // indirect
	github.com/jbenet/goprocess v0.1.4 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/koron/go-ssdp v0.0.4 // indirect
	github.com/libp2p/go-buffer-pool v0.1.0 // indirect
	github.com/libp2p/go-cidranger v1.1.0 // indirect
	github.com/libp2p/go-flow-metrics v0.1.0 // indirect
	github.com/libp2p/go-libp2p-asn-util v0.3.0 // indirect
	github.com/libp2p/go-mplex v0.7.0 // indirect
	github.com/libp2p/go-nat v0.2.0 // indirect
	github.com/libp2p/go-netroute v0.2.1 // indirect
	github.com/libp2p/go-reuseport v0.3.0 // indirect
	github.com/libp2p/go-yamux/v4 v4.0.1 // indirect
	github.com/marten-seemann/tcp v0.0.0-20210406111302-dfbc87cc63fd // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/miekg/dns v1.1.55 // indirect
	github.com/mikioh/tcpinfo v0.0.0-20190314235526-30a79bb1804b // indirect
	github.com/mikioh/tcpopt v0.0.0-20190314235656-172688c1accc // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multiaddr-dns v0.3.1 // indirect
	github.com/multiformats/go-multiaddr-fmt v0.1.0 // indirect
	github.com/multiformats/go-multibase v0.2.0 // indirect
	github.com/multiformats/go-multicodec v0.9.0 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/multiformats/go-multistream v0.4.1 // indirect
	github.com/multiformats/go-varint v0.0.7 // indirect
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417 // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.14.0
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/raulk/go-watchdog v1.3.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/tklauser/go-sysconf v0.3.5 // indirect
	github.com/tklauser/numcpus v0.2.2 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.12.0 // indirect
	golang.org/x/exp v0.0.0-20230713183714-613f0c0eb8a1
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/net v0.14.0 // indirect
	golang.org/x/sys v0.11.0
	golang.org/x/time v0.0.0-20220922220347-f3bd1da661af // indirect
	golang.org/x/tools v0.12.1-0.20230818130535-1517d1a3ba60 // indirect
	google.golang.org/protobuf v1.31.0
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	lukechampine.com/blake3 v1.2.1 // indirect
)
