#./build/waku --rpc=true --rpc-port=8548  --max-connections=10 --discv5-discovery --discv5-bootstrap-node="enr:-M-4QLOTEs_ZFxCb09FgIezZd5KeTru5CWWyEtMWMN-yUABrerWxckU-pMIh3yO8VjxHpgZ4jU2WSXuK3goW4uYb6c4BgmlkgnY0gmlwhI_G-a6KbXVsdGlhZGRyc7EALzYobm9kZS0wMS5kby1hbXMzLnN0YXR1cy5wcm9kLnN0YXR1c2ltLm5ldAYBu94DiXNlY3AyNTZrMaECoVyonsTGEQvVioM562Q1fjzTb_vKD152PPIdsV7sM6SDdGNwgnZfg3VkcIIjKIV3YWt1MgM"
make build
./build/waku --staticnode "/dns4/node-01.do-ams3.go-waku.test.statusim.net/tcp/30303/p2p/16Uiu2HAm9vnvCQgCDrynDK1h7GJoEZVGvnuzq84RyDQ3DEdXmcX7" --nodekey "299b6e27eaa83596be9896baa7e976a35841e2102138b4753a55335c4f69f80b" --tcp-port=60003 --rpc=true --rpc-port=8548 --ext-ip=122.171.21.147 --max-connections=20 --discv5-discovery --discv5-enr-auto-update --discv5-bootstrap-node="enr:-M-4QLOTEs_ZFxCb09FgIezZd5KeTru5CWWyEtMWMN-yUABrerWxckU-pMIh3yO8VjxHpgZ4jU2WSXuK3goW4uYb6c4BgmlkgnY0gmlwhI_G-a6KbXVsdGlhZGRyc7EALzYobm9kZS0wMS5kby1hbXMzLnN0YXR1cy5wcm9kLnN0YXR1c2ltLm5ldAYBu94DiXNlY3AyNTZrMaECoVyonsTGEQvVioM562Q1fjzTb_vKD152PPIdsV7sM6SDdGNwgnZfg3VkcIIjKIV3YWt1MgM" --store=true --storenode="/dns4/node-01.do-ams3.go-waku.test.statusim.net/tcp/30303/p2p/16Uiu2HAm9vnvCQgCDrynDK1h7GJoEZVGvnuzq84RyDQ3DEdXmcX7" --rpc-admin=true
#./build/waku --staticnode "/ip4/127.0.0.1/tcp/60002/p2p/16Uiu2HAm9eaKoXtVLHHkzs9izcXV9N5fuMwpQk4fsHFgvLyt5zwa" --nodekey "299b6e27eaa83596be9896baa7e976a35841e2102138b4753a55335c4f69f80b" --tcp-port=60003 --rpc=true --rpc-port=8548 --nat=none
