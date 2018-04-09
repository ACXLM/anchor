### DaoCloud Enterprise Anchor 模块

#### 序言
Anchor 模块旨在帮助用户实现应用固定IP的功能。用户可以通过已经分配到的IP与需要分配IP的应用绑定，从而实现应用固定IP的功能。

#### 部署
在模块管理中，选择手动安装模块，从互联网安装，选择daocloud.io/anchor镜像，选择最新版本，安装启用即可。或将daocloud.io/anchor镜像上传至镜像空间，在模块管理中，选择手动安装模块，从镜像工厂安装，选择上传的anchor镜像，选择版本号后，点击安装启用即可。

### 使用
**1、分配IP**

在导航栏中，点击进入地址池管理插件，在暂未分配中，管理员为租户分配能够使用的IP，并在网管子网中加入相应子网所对应的网关。

**2、查找IP对应关系**

普通用户登录后，可在地址池管理插件的IP地址池中看到该用户下所有租户已使用的IP和POD的对应关系。

**3、使用IP**

普通用户选择租户后，可见该租户可分配的IP，用户在部署应用时，将需要的IP地址填入，即可部署带有固定IP的POD。

### Example
**实验环境：**

本次测试集群IP为192.168.4.216

**一、下载源码**

```
git clone git@github.com:DaoCloud/anchor.git
cd anchor/governor/backend
```

**二、构建，推送镜像**

```
docker build -t daocloud.io/anchor:v0.1 .
docker tag daocloud.io/anchor:v0.1 192.168.4.216/daocloud/anchor:v0.1
docker push 192.168.4.216/daocloud/anchor:v0.1
```
**三、部署anchor**

模块中心->模块管理->手动安装模块->从镜像工厂安装->选择192.168.4.216/daocloud/anchor，版本为v0.1，点击安装即可。

![Alt text](https://github.com/ACXLM/Picture/blob/master/anchor/%E5%AE%89%E8%A3%85.png)

**四、管理员分配IP到租户**

进入地址池管理->暂未分配->新增IP，输入IP的起止范围，若只需要一个IP则起止填相同的即可，选择将IP分配给相应租户。

![Alt text](https://github.com/ACXLM/Picture/blob/master/anchor/%E6%96%B0%E5%A2%9EIP.png)
![Alt text](https://github.com/ACXLM/Picture/blob/master/anchor/%E6%96%B0%E5%A2%9EIP_List.png)

**五、管理员为子网设置网管**

进入网关子网->新增网关子网，输入子网及其网关，点击确认即可。

![Alt text](https://github.com/ACXLM/Picture/blob/master/anchor/%E6%96%B0%E5%A2%9E%E7%BD%91%E5%85%B3.png)

**六、使用IP**

租户在部署应用时，可以输入已分配给自己的IP部署应用。然后可在地址池管理->IP地址池中看到应用的相关信息。

