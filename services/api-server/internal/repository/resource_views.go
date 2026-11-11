package repository

const (
	ActiveSurveys     Statement = `SELECT s.id,s.title,s.description,s.survey_type,s.start_date,s.end_date FROM survey s WHERE s.is_active=1 AND datetime(s.start_date)<=datetime('now') AND datetime(s.end_date)>=datetime('now') AND NOT EXISTS(SELECT 1 FROM user_survey u WHERE u.user_id=? AND u.survey_id=s.id AND u.completed=1) ORDER BY s.id`
	VideoDefinitions  Statement = `SELECT name,duration,location,COALESCE(subtitle_base,''),COALESCE(has_translations,0) FROM video_definition ORDER BY id`
	VideoSchedules    Statement = `SELECT channel,event,COALESCE(preshow_seconds,0),COALESCE(playlist,'[]'),COALESCE(loop,0),COALESCE(gap_seconds,0),COALESCE(postshow_seconds,0),COALESCE(silence_please,0) FROM video_schedule ORDER BY id`
	TrustActivities   Statement = `SELECT activity_id,activity_name,description,activity_type,COALESCE(required_level,1),COALESCE(reward_points,0) FROM trust_activity_def WHERE is_active=1 ORDER BY id`
	UserTrustGroups   Statement = `SELECT g.group_id,g.group_name,g.description,COALESCE(g.min_trust_level,0),COALESCE(g.max_members,50),(SELECT count(*) FROM user_trust_group m WHERE m.group_id=g.group_id AND m.is_active=1),COALESCE(u.trust_level,1),g.created_at,u.joined_at FROM user_trust_group u JOIN trust_group_def g ON g.group_id=u.group_id WHERE u.user_id=? AND u.is_active=1 AND g.is_active=1 ORDER BY u.id`
	ActiveInfractions Statement = `SELECT COALESCE(severity,'low') FROM infraction WHERE user_id=? AND status='active' AND (end_time IS NULL OR datetime(end_time)>datetime('now'))`
)
