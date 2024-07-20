The [fly.io distributed system challanges](https://fly.io/dist-sys/) solved in go.

# Explanations

## Unique ID Generation

For generating a globally unique ID, we use the name of each node which is given
on initalization, also each node keeps track of a counter value which is 
incremented, on each generate request.

We concatenate these two, and we get the globally unique ID that we send back. 


## Broadcast challenges

- **b)**: Because we know that there will be no network partitions we can
 assume that all messages will be delivered. When a node recieves a broadcast
 it checks if it came from another node or not, if another node sent it, then
 it doesn't need to broadcast it further, else it needs to sent to all others.

- **c)**: Now there is no guarantee that each node will actually get the 
  messages between each other. So we need to keep track of which message was
  actually received by which node.
  Each node does the same:
    1. When a broadcast is received, it is added to the list of recieved messages.
      If it didn't came from another node, then it is to be distributed to all.
    1. The messages that are to be distributed are only marked as sent after 
      the destination node confirms that it recieved it.

- **d)**: The previous solution is working fine in this case too, the only
  change is that the distribution of messages are done less frequently.

- **e)**: In this case to reduce the number of messages between nodes, we batch
  them together.


## Grow-Only Counter

To make our task easier we can rely on a sequentially-consistent key/value store
service provided by Maelstrom. 

*Sequential Consistency: A process in a sequentially consistent system may be*
*far ahead, or behind, of other processes. For instance, they may read*
*arbitrarily stale state. However, once a process A has observed some operation*
*from process B, it can never observe a state prior to B.*

From our end only eventual consistency *(ordering does not matter)* is required.
Every time we recieve an `add` request we need to add the provided delta to a
global counter *(stored in the key/value store)*. For every `read` request we
need to return the current count *(not exactly)*.


Our problem here is illustrated in the following scenario:
```
Key/Value -----X--------X---------X---|
               |        |         |   
              Read      |       Write 
               |        |         |   
        A -----+--------|---------+---|
                        |             
                      Write           
                        |             
        B --------------+-------------|
```

---

1. This problem arrises when we have a single global counter, because other
  nodes can also make changes to it we need to read it every time we want to
  write to it. And between that read and write operation another write is 
  possible from another node.
  So in this case we need to use the `CompareAndSwap` update.
  Cost:
    * read: `1`
    * write: `2 + possible conflicts`

1. To avoid this problem we can keep a counter for every node, every node
  knows the value of its own counter but not other nodes' counters. The
  key/value store would look like this: `n1 -> 413, n2 -> 4619, n3 -> 5431`.
  To answear the read request every time we would need to read all other
  counters.
  Cost:
    * read: `node count - 1`
    * write: `1`

**Which option is better?**
  The answear is dependent on what do you want, greater read or write throughput.
  With the first one you get faster reads, only a single key needs to be read, 
  for the second option we would get faster writes but for read requests all 
  other nodes' keys would need to be read.

I choose the first option, because either way nodes would periodically need to
poll for changes, now after this it's just a `CompareAndSwap` to perform the
update, if it fails we retry after the next read.
