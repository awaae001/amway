package client

import (
	"amway/config"
	rolePb "amway/grpc/gen/role_center"
	"context"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"
)

// AssignRole 根据提供的 guildID, configID 和 userID 分配一个身份组。
// 它会检查配置的时间限制，除非 debug 模式被启用。
func (c *GRPCClient) AssignRole(guildID, configID, userID string) (bool, error) {
	// 检查是否连接到服务器
	if !c.IsConnected() {
		return false, fmt.Errorf("未连接到服务器")
	}

	// 从配置中查找角色信息
	guildRoles, ok := config.Cfg.RoleConfig[guildID]
	if !ok {
		return false, fmt.Errorf("未找到 guild_id '%s' 的角色配置", guildID)
	}

	roleDetail, ok := guildRoles[configID]
	if !ok {
		return false, fmt.Errorf("未找到 config_id '%s' 的角色配置", configID)
	}

	// 检查时间锁，除非 debug 模式开启
	if !config.Cfg.Debug {
		now := time.Now().Unix()
		startAt, err := strconv.ParseInt(roleDetail.StartAt, 10, 64)
		if err != nil {
			return false, fmt.Errorf("解析 start_at 时间失败: %w", err)
		}
		endAt, err := strconv.ParseInt(roleDetail.EndAt, 10, 64)
		if err != nil {
			return false, fmt.Errorf("解析 end_at 时间失败: %w", err)
		}

		if now < startAt || now > endAt {
			return false, fmt.Errorf("当前时间不在允许的时间范围内")
		}
	}

	// 构建 gRPC 请求
	req := &rolePb.AssignRoleRequest{
		UserId:  userID,
		GuildId: guildID,
		RoleId:  roleDetail.GRPCConfig.RoleID,
		//OperatorId: "",  取消，一般不提供
	}

	// 序列化请求
	requestPayload, err := proto.Marshal(req)
	if err != nil {
		return false, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 设置请求超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 通过网关发送请求
	gatewayResp, err := c.SendGatewayRequest(ctx, "/role_center.RoleService/AssignRole", requestPayload, 5)
	if err != nil {
		return false, fmt.Errorf("调用 AssignRole gRPC 服务失败: %w", err)
	}

	// 反序列化响应
	var resp rolePb.AssignRoleResponse
	err = proto.Unmarshal(gatewayResp.Payload, &resp)
	if err != nil {
		return false, fmt.Errorf("反序列化响应失败: %w", err)
	}

	// 检查响应
	if !resp.Success {
		return false, fmt.Errorf("分配角色失败: %s", resp.Message)
	}

	return true, nil
}
