package zentao

import (
	"testing"

	"workbench/internal/config"
)

func TestURLPathInfoWhenConfigured(t *testing.T) {
	SetConfig(config.ZentaoConfig{URL: "http://10.211.55.4:8080", RequestType: "PATH_INFO"})
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"demand", DemandViewURL(2215), "http://10.211.55.4:8080/demand-view-2215.html#app=demandpool"},
		{"demand_clarify", DemandClarifyURL(2215), "http://10.211.55.4:8080/demand-clarify-2215.html"},
		{"story", StoryViewURL(72111), "http://10.211.55.4:8080/story-view-72111.html#app=project"},
		{"task", TaskViewURL(190651), "http://10.211.55.4:8080/task-view-190651.html"},
		{"bug", BugViewURL(39501), "http://10.211.55.4:8080/bug-view-39501.html"},
		{"review", URL("review", "view", "reviewID=1"), "http://10.211.55.4:8080/review-view-1.html"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Fatalf("%s = %s, want %s", c.name, c.got, c.want)
		}
	}
}

func TestURLGetWhenConfigured(t *testing.T) {
	SetConfig(config.ZentaoConfig{URL: "http://10.211.55.4:8080", RequestType: "GET"})
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
	SetConfig(config.ZentaoConfig{URL: "http://10.211.55.4:8080", RequestType: "GET"})

	u1 := StoryViewURL(12345)
	want1 := "http://10.211.55.4:8080/index.php?m=story&f=view&storyID=12345&id=12345#app=project"
	if u1 != want1 {
		t.Fatalf("StoryViewURL(12345) = %s, want %s", u1, want1)
	}

	u2 := URL("story", "view", "id=12345")
	want2 := "http://10.211.55.4:8080/index.php?m=story&f=view&id=12345&storyID=12345#app=project"
	if u2 != want2 {
		t.Fatalf("URL(story, view, id=12345) = %s, want %s", u2, want2)
	}

	if u0 := StoryViewURL(0); u0 != "" {
		t.Fatalf("StoryViewURL(0) = %q, want empty", u0)
	}
}

func TestStoryViewURLPathInfoHash(t *testing.T) {
	SetConfig(config.ZentaoConfig{URL: "http://127.0.0.1:8080", RequestType: "PATH_INFO"})
	got := StoryViewURL(12345)
	want := "http://127.0.0.1:8080/story-view-12345.html#app=project"
	if got != want {
		t.Fatalf("StoryViewURL PATH_INFO = %s, want %s", got, want)
	}
}
