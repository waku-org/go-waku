a=0

declare -a pids 

while [ $a -lt 20 ]
do
   echo $a
   a=`expr $a + 1`
   ./build/waku --max-connections=50 --staticnode="/ip4/127.0.0.1/tcp/60003/p2p/16Uiu2HAmDfKacf97WiQyye4aQFKPjZJGaQiRUoZjzPztsfDFBfPG" &
   #./build/waku --max-connections=10 --store --storenode="/ip4/127.0.0.1/tcp/60003/p2p/16Uiu2HAmDfKacf97WiQyye4aQFKPjZJGaQiRUoZjzPztsfDFBfPG" &
   #./build/waku --staticnode "/ip4/127.0.0.1/tcp/60003/p2p/16Uiu2HAmDfKacf97WiQyye4aQFKPjZJGaQiRUoZjzPztsfDFBfPG" --max-connections=10 --store --storenode="/ip4/127.0.0.1/tcp/60003/p2p/16Uiu2HAmDfKacf97WiQyye4aQFKPjZJGaQiRUoZjzPztsfDFBfPG" &
done



#./build/waku --staticnode "/ip4/127.0.0.1/tcp/60002/p2p/16Uiu2HAm9eaKoXtVLHHkzs9izcXV9N5fuMwpQk4fsHFgvLyt5zwa" --nodekey "299b6e27eaa83596be9896baa7e976a35841e2102138b4753a55335c4f69f80b" --tcp-port=60003 --rpc=true --rpc-port=8548 --nat=none
