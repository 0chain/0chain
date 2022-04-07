
miner=miner$1
redis=redis_$1
redis_txns=redis_txns_$1

root=$(pwd)

[ -d $root/data/$miner/data/redis/state ] || mkdir -p $root/data/$miner/data/redis/state
[ -d $root/data/$miner/data/redis/transactions ] || mkdir -p $root/data/$miner/data/redis/transactions


num=$(docker ps -a --filter "name=^${redis}$" | wc -l)

echo -n "[1/4] remove $redis: "
[ $num -eq 2 ] && docker rm $redis --force || echo " SKIPPED"

echo -n "[2/4] install $redis: " && \
docker run --name $redis \
--restart always -p 63${1}0:6379 \
-v $root/data/$miner/config:/0chain/config \
-v  $root/data/$miner/data:/0chain/data \
-d redis:alpine redis-server /0chain/config/redis/state.redis.conf


num=$(docker ps -a --filter "name=^${redis_txns}$" | wc -l)


echo -n "[3/4] remove $redis_txns: "
[ $num -eq 2 ] && docker rm $redis_txns --force || echo " SKIPPED"

echo -n "[4/4] install $redis_txns: " && \
docker run --name $redis_txns \
--restart always -p 63${1}1:6379 \
-v $root/data/$miner/config:/0chain/config \
-v  $root/data/$miner/data:/0chain/data \
-d redis:alpine redis-server /0chain/config/redis/transactions.redis.conf
