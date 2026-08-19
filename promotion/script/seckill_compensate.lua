local purchase_num = tonumber(ARGV[1])
if purchase_num <= 0 then
    return -1
end

if redis.call('EXISTS', KEYS[1]) == 0 then
    return -2
end

if KEYS[3] and KEYS[3] ~= '' then
    local stock_ttl = redis.call('TTL', KEYS[1])
    local marked
    if stock_ttl > 0 then
        marked = redis.call('SET', KEYS[3], '1', 'NX', 'EX', stock_ttl)
    else
        marked = redis.call('SET', KEYS[3], '1', 'NX')
    end
    if marked == false then
        return -3
    end
end

redis.call('HINCRBY', KEYS[1], 'stock', purchase_num)

local purchased_num = tonumber(redis.call('GET', KEYS[2]) or '0')
local remaining_num = purchased_num - purchase_num
if remaining_num > 0 then
    redis.call('SET', KEYS[2], remaining_num, 'KEEPTTL')
else
    redis.call('DEL', KEYS[2])
    remaining_num = 0
end

return remaining_num
