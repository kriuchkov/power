# Client <-> server over UDP and TCP pow protocols

Denial-of-Service-attacks are a typical situation when providing services over a network. A method for preventing DoS-attacks is to have the client show its dedication towards the service before gaining access to it. As proof of dedication, the client is requested to compute an answer to an algorithmic nonce.

The nonce should be hard to solve but the answer should be easy to verify. When the computation is done, the answer is sent to the server which verifies the answer. The nonce puts a heavy load on the client if several requests are made in a short time span; this prevents the client from abusing the service. This way of authentication for using a service is named a proof-of-work protocol. The client will have to provide proof of work (POW).

POW will not suffice as a guardian against DDoS. DoS protection is perhaps more essential because any client on the Internet is able to perform DoS attacks on their own. Having your service use an implementation of POW could not aid with handling DoS attacks if you use a protocol such as **TCP** (Transmission Control Protocol) because you are vulnerable to attacks from any single computer. Because before POW the client sends an SYN-packet to the server telling the server it wants to connect.

The server responds with an **SYN-ACK**. In the end, the client responds with an **ACK** back to the server. Only after this, a connection has been established and data can be sent. A typical DoS attack is simply to flood a server with **SYN**-packets but never respond to the servers **SYN-ACK**-packets. The server can have a limited number of outstanding SYN-ACKs and when the queue is full it will drop incoming **SYN**s.

# Implementation

This version of a POW that I implemented over **UDP** and **TCP** had the following characteristics:

1. The protocol needs to scale well when more clients try to connect
2. The workload on the server should be lower than on the client
3. It should be impossible for a client to do any precalculations

## Reverse Computing a Hash

The pow-alogorith explained below is based on the reverse computation of hashed bytes from DOS-resistant authentication. Generally, it is almost impossible to reverse a hash to original bytes, however, it is easier to find a similar hash easy. 

This is the key idea for this puzzle:  
```
h(X) = 000 . . . 000 | {z } m zeros +Y
```

where h is a hash function, m ∈ Z.  

The difficulty m specifies how many leading zeros the hash h should contain. The client then attempts to find an X that has a hash value with the set number of leading zero.

## The protocol
![protocol](https://www.planttext.com/api/plantuml/png/VP6x2iCm34LtVuL6P_0FX582NJjtDxQQ1Fm8bbB8tzSc8Qy-Di4zEbV63R5EF7edHZlSOWYWhf37UqySCDLbhk61gNzE4lt0KoMs6DHCbyK32hBJr5NYhxK4Q1WaHVT2MwtmHQcVj6GpWBOs8T7HduFv3fDGCu86Cw_qCOWbNBZLt29dpcUNRd65Il-U8Wps2tPo6HVfrBf_q7RU9zVaWlm7Rm00)

A request with a hash is sent to the server over **UDP** the protocol. The hash is a SHA256[32] hash made up of nonce. The server sends the nonce to the client, the client has to solve that when the solution sends back to the server.  A common property of the hashes is that they have to be non-pre-computable. If a hash is pre-computable, a hacker could spend some time calculating solutions for a hash before an attack.

## How to check using Docker

```
make docker-run
```
