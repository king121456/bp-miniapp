#!/bin/bash
B=${B:-http://localhost:8080}; P=0; F=0
chk(){ if [ "$2" = "$3" ]; then echo "PASS $1"; P=$((P+1)); else echo "FAIL $1 (got $2, want $3)"; F=$((F+1)); fi; }
req(){ curl -s -o /tmp/r -w '%{http_code}' -X "$1" "$B$2" -H "Authorization: Bearer $3" -H 'Content-Type: application/json' ${4:+-d "$4"}; }
C='{"agreedAt":"2026-10-11T19:00:00+08:00","version":"1.0"}'
chk "1 未同意登录被拒" $(req POST /api/auth/login "" '{"code":"pX"}') 403
req POST /api/auth/login "" "{\"code\":\"p1\",\"nickname\":\"张阿姨\",\"consent\":$C}" >/dev/null; PT=$(jq -r .data.token /tmp/r)
req POST /api/auth/login "" "{\"code\":\"f1\",\"consent\":$C}" >/dev/null; FT=$(jq -r .data.token /tmp/r)
chk "未带token被拒" $(req GET /api/bp "") 401
chk "录入血压" $(req POST /api/bp $PT '{"sys":135,"dia":85,"hr":72,"tags":["头晕"],"note":"早上起床后"}') 200
BID=$(jq -r .data.record.id /tmp/r); chk "无授权时返回needSubscribe" $(jq -r .data.needSubscribe.kind /tmp/r) measure
chk "明显输错被拦" $(req POST /api/bp $PT '{"sys":80,"dia":120}') 400
chk "编辑血压" $(req PUT /api/bp/$BID $PT '{"sys":130,"dia":82,"hr":70,"note":"改过"}') 200
chk "统计接口" $(req GET '/api/bp/stats?range=week' $PT) 200; chk "均值" $(jq -r .data.average.sys /tmp/r) 130
req POST /api/meds $PT '{"name":"氨氯地平","dose":"5mg","times":["08:00","20:00"]}' >/dev/null; MID=$(jq -r .data.id /tmp/r)
chk "3 打卡08:00" $(req POST /api/meds/$MID/checkin $PT '{"time":"08:00"}') 200
req GET /api/meds $PT >/dev/null
chk "08:00已打" $(jq -r '.data[0].slots[0].done' /tmp/r) true
chk "20:00未受影响" $(jq -r '.data[0].slots[1].done' /tmp/r) false
chk "整药未完成" $(jq -r '.data[0].done' /tmp/r) false
chk "无效时间点被拒" $(req POST /api/meds/$MID/checkin $PT '{"time":"09:00"}') 400
req POST /api/subscribe $PT '{"accepted":["med","med"]}' >/dev/null
req POST /api/subscribe $PT '{"accepted":["med"]}' >/dev/null; chk "授权次数累加" $(jq -r .data.med /tmp/r) 3
req POST /api/meds/$MID/checkin $PT '{"time":"20:00"}' >/dev/null; chk "4 次数>=2不提示" $(jq -r .data.needSubscribe /tmp/r) null
req POST /api/family/invite $PT >/dev/null; CODE=$(jq -r .data.code /tmp/r)
chk "家人绑定" $(req POST /api/family/bind $FT "{\"code\":\"$CODE\"}") 200
chk "邀请码只能用一次" $(req POST /api/family/bind $FT "{\"code\":\"$CODE\"}") 404
PID=$(jq -r .data.patientId <<<"$(curl -s $B/api/me -H "Authorization: Bearer $PT")"); PID=$(curl -s $B/api/me -H "Authorization: Bearer $PT" | jq -r .data._id)
chk "家人只读查看" $(req GET /api/family/patients/$PID/bp $FT) 200; chk "家人看到1条" $(jq '.data|length' /tmp/r) 1
chk "2 家人改患者血压被拒" $(req PUT /api/bp/$BID $FT '{"sys":120,"dia":80}') 404
chk "2 家人删患者血压被拒" $(req DELETE /api/bp/$BID $FT) 404
chk "2 家人给患者药打卡被拒" $(req POST /api/meds/$MID/checkin $FT '{"time":"08:00"}') 404
chk "2 家人改患者药被拒" $(req PUT /api/meds/$MID $FT '{"name":"x","times":["08:00"]}') 404
req GET /api/bp $FT >/dev/null; chk "家人自己的列表里没有患者数据" $(jq '.data|length' /tmp/r) 0
req GET /api/family $PT >/dev/null; LID=$(jq -r '.data.members[0].id' /tmp/r)
chk "解绑" $(req DELETE /api/family/$LID $PT) 200
chk "2 解绑后家人看不到" $(req GET /api/family/patients/$PID/bp $FT) 403
chk "6 报告默认30天" $(req GET /api/report $PT) 200
chk "报告天数" $(jq -r .data.days /tmp/r) 30
chk "报告完成率" $(jq -r .data.adherence.taken /tmp/r) 2
chk "报告症状" $(jq -r '.data.symptoms|length' /tmp/r) 0
chk "报告超90天被拒" $(req GET '/api/report?from=2026-01-01&to=2026-06-01' $PT) 400
chk "报告非法days被拒" $(req GET '/api/report?days=14' $PT) 400
req GET /api/account/export $PT >/dev/null
chk "5 导出备注为原文" "$(jq -r '.data.bloodPressure[0].note' /tmp/r)" "改过"
chk "5 导出药名为原文" "$(jq -r '.data.medications[0].name' /tmp/r)" "氨氯地平"
chk "注销" $(req DELETE /api/account $PT) 200
chk "注销后token失效" $(req GET /api/bp $PT) 401
echo "PASS=$P FAIL=$F"
