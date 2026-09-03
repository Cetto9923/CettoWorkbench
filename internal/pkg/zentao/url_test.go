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
		{"story", StoryViewURL(72111), "http://10.211.55.4:8080/index.php?m=story&f=view&storyID=72111&id=72111"},
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
