"""Map server-advertised resource names to known legacy HTTP registrations."""
import ast
import json
from pathlib import Path

root=Path(__file__).resolve().parents[2]
source=root.parent/'LaServer/services/game-api/business/account'
routes=json.loads((root/'json/api/legacy-routes.json').read_text(encoding='utf-8'))
known={r['path']:r for r in routes}
mapping={
 'device_capabilities':'/account/device_capabilities/lookup', 'userdata':'/account/userdata',
 'set_friends':'/account/get_friends','set_friend_statues':'/account/get_friend_statues','online_friends':'/account/get_online_friends',
 'set_rank':'/account/get_rank','currency':'/account/get_currency','forge_rates':'/account/get_forge_rates','motd':'/account/get_motd',
 'outfit_defs':'/account/get_outfit','achievements':'/account/get_achievements','achievement_stats':'/account/get_achievement_stats',
 'relationship_defs':'/account/get_relationship_abilities','questionnaires':'/account/get_questionnaires','event_schedule':'/account/get_event_schedule',
 'set_world_quests':'/account/get_account_world_quests','iap_list':'/account/iaplist','invite_list':'/account/get_invites',
 'unlocks':'/account/get_unlocks','get_shop':'/account/get_shop','spirit_shops':'/account/get_spirit_shops','collectibles':'/account/get_collectibles',
 'wing_buffs':'/account/wing_buffs/get','gift_messages':'/account/get_pending_messages','set_app_badge_number':'/account/get_app_badge_number',
 'get_consumable_defs':'/account/consumable/get_consumable_defs','get_consumables':'/account/consumable/get_consumables',
 'get_buff_defs':'/account/buff/get_buff_defs','get_buffs':'/account/buff/get_buffs','get_lootboxes':'/account/lootboxes/get',
 'checkpoints':'/account/get_checkpoints','generic_shop_defs':'/account/get_generic_shops','external_account_friends':'/account/get_external_friends',
 'serendipity_matches':'/account/serendipity/get','get_star_tag_defs':'/account/star/get',
 'status_unlocks':'/service/status/api/v1/get_unlocks','free_gifts':'/service/relationship/api/v1/free_gifts/get_pending',
 'get_map_defs':'/account/get_map_defs','achievement_defs':'/account/get_achievement_defs','achievement_stats_tracking':'/account/get_achievement_stats_tracking',
 'event_currency_defs':'/account/get_event_currency_defs','infractions':'/service/infraction/api/v1/get_infractions_status_decorated',
 'messages':'/service/message/api/v2/get_all',
}
resources={}
for relative in ['create/create.py','login/login.py','auth_create/auth_create.py','auth_login/auth_login.py']:
 tree=ast.parse((source/relative).read_text(encoding='utf-8-sig'))
 names=next(ast.literal_eval(v) for n in ast.walk(tree) if isinstance(n,ast.Dict) for k,v in zip(n.keys,n.values) if isinstance(k,ast.Constant) and k.value=='resource_list')
 for name in names:resources.setdefault(name,[]).append(relative)
out=[]
for name,advertised in resources.items():
 path=mapping.get(name);route=known.get(path)
 out.append({'resource':name,'advertised_by':advertised,'candidate_path':path,'mapping_evidence':'legacy server handler/response naming; client request trace pending' if path else 'unresolved','implementation':route['status'] if route else 'unresolved','requested_result':501 if route and route['status']=='pending' else 'requires_runtime_check','blocks_entire_login':'unverified','client_verified':False})
(root/'json/api/login-dependencies.json').write_text(json.dumps(out,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
lines=['# 登录资源依赖基线','', '从四种旧认证响应的 resource_list 提取，不代表每个客户端都会请求全部资源。路径是根据服务端名称建立的候选映射；实际请求顺序、版本和阻断点必须以客户端记录验证，不能把 501 自动等同于整次登录失败。', '', '| 资源名 | 候选请求路径 | 当前实现 |', '| --- | --- | --- |']
for item in out:lines.append('| `'+item['resource']+'` | '+('`'+item['candidate_path']+'`' if item['candidate_path'] else '待定位')+' | '+item['implementation']+' |')
lines+=['', '优先补齐：装扮 → 解锁/光翼 → 好友/邀请 → 任务 → 魔法和 buff 定义。该顺序是开发建议，不是已观测的客户端请求顺序。', '', 'set_friend_status、set_random、external_links、get_user_profile、emote_sequences 等无法由现有路径直接证明的资源保留 unresolved；不能根据相似名字擅自返回成功。']
(root/'docs/login-dependencies.md').write_text('\n'.join(lines)+'\n',encoding='utf-8')
print(f'{len(out)} advertised resources; {sum(x["implementation"]=="unresolved" for x in out)} mappings unresolved')
