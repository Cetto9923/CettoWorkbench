package zentao

import (
	"testing"

	"workbench/internal/config"
)

func TestURLTempCheck(t *testing.T) {
	SetConfig(config.ZentaoConfig{URL: "http://10.211.55.4:8080", RequestType: "PATH_INFO"})
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"demand", DemandViewURL(2215), "http://10.211.55.4:8080/index.php?m=demand&f=view&demandID=2215&id=2215#app=demandpool"},
		{"demand_clarify", DemandClarifyURL(2215), "http://10.211.55.4:8080/index.php?m=demand&f=clarify&demandID=2215&id=2215"},
		{"story", StoryViewURL(72111), "http://10.211.55.4:8080/index.php?m=story&f=view&storyID=72111&id=72111#app=project"},
		{"task", TaskViewURL(190651), "http://10.211.55.4:8080/index.php?m=task&f=view&taskID=190651&id=190651"},
		{"bug", BugViewURL(39501), "http://10.211.55.4:8080/index.php?m=bug&f=view&bugID=39501&id=39501"},
		{"review", URL("review", "view", "reviewID=1"), "http://10.211.55.4:8080/index.php?m=review&f=view&reviewID=1&id=1"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Fatalf("%s = %s, want %s", c.name, c.got, c.want)
		}
	}
}

func TestStoryViewURLParamsAndHash(t *testing.T) {
	SetConfig(config.ZentaoConfig{URL: "http://10.211.55.4:8080"})

	// 1. StoryViewURL 必须带 storyID 与 id 两个参数，且以 #app=project 结尾
	u1 := StoryViewURL(12345)
	want1 := "http://10.211.55.4:8080/index.php?m=story&f=view&storyID=12345&id=12345#app=project"
	if u1 != want1 {
		t.Fatalf("StoryViewURL(12345) = %s, want %s", u1, want1)
	}

	// 2. 传 id= 时自动补齐 storyID 与 #app=project
	u2 := URL("story", "view", "id=12345")
	want2 := "http://10.211.55.4:8080/index.php?m=story&f=view&id=12345&storyID=12345#app=project"
	if u2 != want2 {
		t.Fatalf("URL(story, view, id=12345) = %s, want %s", u2, want2)
	}

	// 3. ID 为 0 时返回空
	if u0 := StoryViewURL(0); u0 != "" {
		t.Fatalf("StoryViewURL(0) = %q, want empty", u0)
	}
}
