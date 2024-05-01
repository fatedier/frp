# Expose local Port - > frp-demonstration

Reference → [https://github.com/REZ-OAN/frp](https://github.com/REZ-OAN/frp)

# Step-1 Go on your vm or bm

Goto your server

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled.png)

# Step-2 Download release and Extract It (on vm or bm)

Download the latest release from the github repo

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(1).png)

Select the one which meet your system architecture

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(2).png)

Now hover on the file and `right click` on it select the `copy link address`

Now `goto` the `terminal` where you `ssh` to the server

go to preferred folder where you want to download the release

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(3).png)

Now using `wget` download the release

```bash
wget https://github.com/REZ-OAN/frp/releases/download/v0.57.0/frp_0.57.0_linux_amd64.tar.gz
```

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(4).png)

In this directory you will see like this

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(5).png)

Now extract the `.tar.gz` file

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(6).png)

Now `goto`   `frp_0.57.0_linux_amd64/` this folder

This files will be there

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(7).png)

# Step-3 Running frp-server  ( on vm or bm)

Now edit the `frps.toml` because we will run the `frp-server` on our `vm` or `bm` 

We need to specify a `port` on our `vm` or `bm` on which the `frp-server`will listen to

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(8).png)

Using `nano` edit the `bindPort` if you want to 

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(9).png)

I have edited the port number

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(10).png)

Now run the server 

```bash
./frps -c ./frps.toml
```

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(11).png)

# Step-4 Running our codeserver-python (on local machine)

First need to pull the image from the docker.hub

```bash
docker pull poridhi/codeserver-python:v1.2
```

Now, run this and copy the image id to run the image

```bash
docker images
```

Run using below command

```bash
docker run -it -p 5000:8080 b25217878034
```

Your terminal will look like this

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(12).png)

# Step-5 Running frp-client (on our local machine)

Firstly do every thing in done in the **Step-2 ,** after doing all of those things 

Edit the `frpc.toml` 

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(13).png)

Here, `serverAddr` refers to the server from where you locally running service will be **exposed**

And `serverPort` refers to the port where the`frp-server` is listening

And `localPort` refers to the port in which our service is **running**

And `remotePort` refers to the `port`in the `remoteServer` in which our service will be **exposed** to 

Edit the `serverPort` to `4848` and `localPort` to `5000` and set `remotePort` to  `7050` using `nano`

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(14).png)

Now run the `frp-client` 

```bash
./frpc -c ./frpc.toml
```

# Step-6 create tunnel on cloudflare and then expose the port

Go to cloudflare dashboard select the `Zero Trust`

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(15).png)

Then open networks drop down and select the tunnels 

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(16).png)

Now click on `create tunnel` 

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(17).png)

Select the recommended connector

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(18).png)

Then click on `next`  and then give a name i choose `test-frp`

Now choose connector environment , i have chosen `docker` 

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(19).png)

Now copy the connector `command` and run it on `detach` mode on your `server` ( in our case 103.174.50.21)

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(20).png)

Now select the domain  and subdomain , and the url server `ip_address:remotePort` 

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(21).png)

Now hit the URL `test-frp.poridhi.io/?folder=/app/`  will see

![Untitled](https://github.com/REZ-OAN/frp/blob/frp-demonstration-with-cloudflare-tunnel/demonstrated_images/Untitled%20(22).png)