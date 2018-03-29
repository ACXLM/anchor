## StaticIP Plugin API

> 有些东西之后可能会有变化，比如增加IP或删除的返回值。


[toc]
### Auth
1. X-DCE-Access-Token
2. Cookie 中的 DCE_TOKEN

### 获取 Static-IP使用关系列表`/api/v1/static_ip [GET]`
**Headers:**
```json
    X-DCE-Access-Token:
```
**Response:**
```json
 {
    "192.168.4.1": {
        "AppName": " ",
        "ContainerId": "2048",
        "PodName": "2048",
        "ServiceName": " ",
        "StaticIp": "192.168.4.1",
        "TenantName": "test"
    },
    "192.168.4.11": {
        "AppName": " ",
        "ContainerId": "2048",
        "PodName": "2048",
        "ServiceName": " ",
        "StaticIp": "192.168.4.11",
        "TenantName": "default"
    }
}
```

**字段：**

| 字段       | 解释                 |
|------------|----------------------|
| AppName       | 应用名称             |
| ContainerId    | 容器ID          |
| PodName  | POD名称              |
| ServiceName     | 服务名称 |
| StaticIp | 静态IP地址 |
| TenantName | 租户名称 |

### 获取 租户列表`/api/v1/get_tenant_name [GET]`

**Headers:**
```json
    X-DCE-Access-Token:
```
**Response:**
   

```json
[
    "admin",
    "default",
    "kube-public",
    "kube-system",
    "test",
    "test1"
]
```


### 获取 租户已使用IP列表`/api/v1/tenant_ip [GET]`

**Headers:**
```json
    X-DCE-Access-Token:
```
**Args:**
TenantName：租户名（必须）
**Response:**
   

```json
 {
    "default": [
        "192.168.4.11",
        "192.168.4.12"
    ]
}
```

**字段：**

| 字段       | 解释                 |
|------------|----------------------|
| 租户名称       | 租户已使用ip列表             |


### 获取 租户已有全部IP列表`/api/v1/tenant_ip [GET]`
**Headers:**
```json
    X-DCE-Access-Token:
```
**Args:**
TenantName：租户名（必须）
All：是否显示租户所有ip（default=False） 1
**Response:**
   
```json
 {
    "default": [
        "192.168.4.10",
        "192.168.4.11",
        "192.168.4.12",
        "192.168.4.13",
        "192.168.4.14",
        "192.168.4.15",
        "192.168.4.16",
        "192.168.4.17",
        "192.168.4.18",
        "192.168.4.19"
    ]
}
```

**字段：**

| 字段       | 解释                 |
|------------|----------------------|
| 租户名称       | 租户全部ip列表             |

### 获取 租户未使用IP列表`/api/v1/tenant_ip [GET]`
**Headers:**
```json
    X-DCE-Access-Token:
```
**Args:**
TenantName：租户名（必须）
Unuse：是否显示租户未使用ip（default=False） 1
**Response:**
   
```json
{
    "default": [
        "192.168.4.18",
        "192.168.4.19",
        "192.168.4.10",
        "192.168.4.13",
        "192.168.4.14",
        "192.168.4.15",
        "192.168.4.16",
        "192.168.4.17"
    ]
}
```

**字段：**

| 字段       | 解释                 |
|------------|----------------------|
| 租户名称       | 租户未ip列表             |

### 给租户创建/分配IP`/api/v1/tenant_ip [POST]`
**Headers:**
```json
	Content-Type：application/json
    X-DCE-Access-Token:
```
**Request:**

```
{
	"StartIp":"192.168.2.3",
	"EndIp":"192.168.2.8",
	"TenantName":"test"
}
```

**Response:**
   
```json
{
   {
    "Status": "Ok"
   }
}
```

**字段：**

| 字段       | 解释                 |
|------------|----------------------|
| StartIp       | 开始IP             |
| EndIp | 结束IP |
| TenantName | 租户名称 |

### 删除 Static-IP`/api/v1/bulk_delete_sip [POST]`
**Headers:**
```json
	Content-Type：application/json
    X-DCE-Access-Token:
```
**Request:**

```
{
	"StaticIps":["192.168.2.5"],
	"TenantName":"test"
}
```

**Response:**
   
```json
{
    "Status": "Ok"
}
```

**字段：**

| 字段       | 解释                 |
|------------|----------------------|
| StaticIps       | 需要删除的IP段            |
| TenantName | 租户名称 |


### 获取Gateway`/api/v1/gateway [GET]`
**Headers:**
```json
	Content-Type：application/json
    X-DCE-Access-Token:
```

**Response:**
   
```json
[
    {
        "Gateway": "192.168.2.254",
        "Subnet": "192.168.2.0/24"
    },
    {
        "Gateway": "192.168.2.254",
        "Subnet": "192.168.2.0/24"
    }
]
```

**字段：**

| 字段       | 解释                 |
|------------|----------------------|
| Gateway       | 网关            |
| Subnet | 子网 |

### 创建Gateway`/api/v1/gateway [POST]`
**Headers:**
```json
	Content-Type：application/json
    X-DCE-Access-Token:
```

**Request:**

```
{
  "Subnet":"192.168.3.0/24",
  "Gateway":"192.168.3.1"
}
```

**Response:**
   
```json
{
    "Gateway": "192.168.5.1",
    "Subnet": "192.168.5.0/24"
}
```

**字段：**

| 字段       | 解释                 |
|------------|----------------------|
| Gateway       | 网关            |
| Subnet | 子网 |

### 删除Gateway`/api/v1/gateway [DELETE]`
**Headers:**
```json
	Content-Type：application/json
    X-DCE-Access-Token:
```

**Request:**

```
{
  "Subnet":"192.168.4.0/24"
}
```

**Response:**
   
```json
{
    "Subnet": "192.168.4.0/24"
}
```

**字段：**

| 字段       | 解释                 |
|------------|----------------------|
| Subnet | 子网 |

### 400错误
```json
{
    "id": "",
    "message": ""
}

带下划线是 id

id:
    - static_ip_already_exist_error -- 创建失败，静态 IP 已存在
    - static_ip_not_exist_error -- 这条还不存在
    - not_authorized_error
        - "Admin required." "需要管理员权限"
    - static_ip_format_error -- "IP 格式不正确"
    - static_ip_range_too_big -- "IP 范围过大"
    - sip_ip_range_err -- "IP范围错误"
    - static_ip_not_belong_to_tenant -- 静态 IP 不属于该租户
```


