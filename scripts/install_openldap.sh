#!/bin/bash

# Fedora OpenLDAP 自动安装和配置脚本

set -e

echo "=== Fedora OpenLDAP 安装和配置脚本 ==="

# 检查是否为root用户
if [[ $EUID -eq 0 ]]; then
   echo "请不要以root用户运行此脚本，请使用有sudo权限的普通用户"
   exit 1
fi

# 设置变量
DOMAIN="example.com"
ADMIN_PASSWORD="secret"
ORGANIZATION="Example Organization"

echo "配置信息："
echo "  域名: $DOMAIN"
echo "  管理员密码: $ADMIN_PASSWORD"
echo "  组织: $ORGANIZATION"
echo ""

read -p "是否继续安装? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "安装已取消"
    exit 1
fi

# 1. 安装软件包
echo "1. 安装 OpenLDAP 软件包..."
sudo dnf install -y openldap-servers openldap-clients openldap

# 2. 启动服务
echo "2. 启动 OpenLDAP 服务..."
sudo systemctl start slapd
sudo systemctl enable slapd

# 等待服务启动
sleep 3

# 3. 生成管理员密码哈希
echo "3. 生成管理员密码哈希..."
ADMIN_HASH=$(sudo slappasswd -s "$ADMIN_PASSWORD")
echo "密码哈希: $ADMIN_HASH"

# 4. 创建基础配置文件
echo "4. 创建配置文件..."

# 基础数据库配置
cat > /tmp/db.ldif << EOF
dn: olcDatabase={2}mdb,cn=config
changetype: modify
replace: olcSuffix
olcSuffix: dc=example,dc=com

dn: olcDatabase={2}mdb,cn=config
changetype: modify
replace: olcRootDN
olcRootDN: cn=admin,dc=example,dc=com

dn: olcDatabase={2}mdb,cn=config
changetype: modify
replace: olcRootPW
olcRootPW: $ADMIN_HASH
EOF

# 监控配置
cat > /tmp/monitor_module.ldif << EOF
dn: cn=module,cn=config
changetype: add
objectClass: olcModuleList
cn: module
olcModulepath: /usr/lib64/openldap
olcModuleload: back_monitor.la
EOF

cat > /tmp/monitor_db.ldif << EOF
dn: olcDatabase={1}monitor,cn=config
changetype: modify
replace: olcRootDN
olcRootDN: cn=monitor,cn=Monitor

dn: olcDatabase={1}monitor,cn=config
changetype: modify
replace: olcAccess
olcAccess: to dn.subtree="cn=Monitor" 
  by dn="cn=admin,dc=example,dc=com" read 
  by * read
EOF

# 基础目录结构
cat > /tmp/base.ldif << EOF
dn: dc=example,dc=com
objectClass: top
objectClass: dcObject
objectClass: organization
o: $ORGANIZATION
dc: example

dn: cn=admin,dc=example,dc=com
objectClass: organizationalRole
cn: admin
description: LDAP administrator
EOF

# 5. 应用配置
echo "5. 应用基础配置..."
sudo ldapmodify -Y EXTERNAL -H ldapi:/// -f /tmp/db.ldif

echo "6. 应用监控配置..."
echo "6.1 添加监控模块..."
sudo ldapmodify -Y EXTERNAL -H ldapi:/// -f /tmp/monitor_module.ldif

echo "6.2 配置监控数据库..."
sudo ldapmodify -Y EXTERNAL -H ldapi:/// -f /tmp/monitor_db.ldif

echo "7. 添加基础目录结构..."
ldapadd -x -D "cn=admin,dc=example,dc=com" -w "$ADMIN_PASSWORD" -f /tmp/base.ldif

# 8. 配置防火墙
echo "8. 配置防火墙..."
if systemctl is-active --quiet firewalld; then
    sudo firewall-cmd --permanent --add-service=ldap
    sudo firewall-cmd --reload
    echo "已开放LDAP端口"
else
    echo "防火墙未运行，跳过防火墙配置"
fi

# 9. 验证安装
echo "9. 验证安装..."

echo "检查服务状态..."
sudo systemctl status slapd --no-pager

echo "检查端口监听..."
sudo ss -tlnp | grep :389

echo "测试匿名连接..."
if ldapsearch -x -H ldap://localhost -b "" -s base > /dev/null 2>&1; then
    echo "✓ 匿名连接成功"
else
    echo "✗ 匿名连接失败"
fi

echo "测试管理员认证..."
if ldapsearch -x -H ldap://localhost -D "cn=admin,dc=example,dc=com" -w "$ADMIN_PASSWORD" -b "dc=example,dc=com" > /dev/null 2>&1; then
    echo "✓ 管理员认证成功"
else
    echo "✗ 管理员认证失败"
fi

echo "测试Monitor查询..."
if ldapsearch -x -H ldap://localhost -D "cn=admin,dc=example,dc=com" -w "$ADMIN_PASSWORD" -b "cn=Monitor" -s base > /dev/null 2>&1; then
    echo "✓ Monitor查询成功"
else
    echo "✗ Monitor查询失败，可能需要手动配置"
fi

# 10. 清理临时文件
echo "10. 清理临时文件..."
rm -f /tmp/db.ldif /tmp/monitor_module.ldif /tmp/monitor_db.ldif /tmp/base.ldif

# 11. 输出连接信息
echo ""
echo "=== 安装完成 ==="
echo "LDAP服务器信息："
echo "  服务器地址: 127.0.0.1"
echo "  端口: 389"
echo "  管理员DN: cn=admin,dc=example,dc=com"
echo "  管理员密码: $ADMIN_PASSWORD"
echo "  基础DN: dc=example,dc=com"
echo "  监控DN: cn=Monitor"
echo ""
echo "测试命令："
echo "  ldapsearch -x -H ldap://localhost -D \"cn=admin,dc=example,dc=com\" -w \"$ADMIN_PASSWORD\" -b \"cn=Monitor\""
echo ""
echo "更新 openldap_exporter 配置："
echo "  server: \"ldap://localhost:389\""
echo "  bind_dn: \"cn=admin,dc=example,dc=com\""
echo "  bind_password: \"$ADMIN_PASSWORD\""
echo ""
