// =============================================================================
// 文件: internal/model/zentao/demand.go
// 模块: 数据模型
// 类型: model
// 职责: 定义业务需求模型字段与表映射。
// 依赖: 无
// =============================================================================

package model

import (
	"time"
)

// ZtDemand 需求表模型
type ZtDemand struct {
	ID                      int        `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Parent                  int        `gorm:"column:parent;default:0" json:"parent"`
	Pool                    int        `gorm:"column:pool;default:0" json:"pool"`
	Module                  int        `gorm:"column:module;default:0" json:"module"`
	Pri                     string     `gorm:"column:pri;type:char(30);default:''" json:"pri"`
	Severity                string     `gorm:"column:severity;type:char(30);not null" json:"severity"`
	Category                string     `gorm:"column:category;type:char(30);default:''" json:"category"`
	Source                  string     `gorm:"column:source;type:char(30);default:''" json:"source"`
	SourceNote              string     `gorm:"column:sourceNote;type:varchar(255);default:''" json:"sourceNote"`
	Name                    string     `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Desc                    string     `gorm:"column:desc;type:longtext;not null" json:"desc"`
	FeedbackBy              string     `gorm:"column:feedbackBy;type:varchar(255);not null" json:"feedbackBy"`
	Email                   string     `gorm:"column:email;type:varchar(255);default:''" json:"email"`
	Mobile                  string     `gorm:"column:mobile;type:char(30);not null" json:"mobile"`
	AssignedTo              string     `gorm:"column:assignedTo;type:char(30);default:''" json:"assignedTo"`
	QD                      string     `gorm:"column:QD;type:char(30);not null;comment:测试负责人" json:"qd"`
	RD                      string     `gorm:"column:RD;type:char(30);not null;comment:验收负责人" json:"rd"`
	BRA                     string     `gorm:"column:BRA;type:char(30);not null;comment:业务需求负责人" json:"bra"`
	MainSystem              string     `gorm:"column:mainSystem;type:char(30);not null;comment:主系统" json:"mainSystem"`
	Reviewer                string     `gorm:"column:reviewer;type:varchar(255);not null" json:"reviewer"`
	Status                  string     `gorm:"column:status;type:char(30);default:''" json:"status"`
	Stage                   string     `gorm:"column:stage;type:enum('wait','distributed','inroadmap','incharter','developing','delivering','delivered','closed');default:'wait'" json:"stage"`
	Hang                    string     `gorm:"column:hang;type:enum('0','1');default:'0';comment:是否挂起" json:"hang"`
	HangUpReason            string     `gorm:"column:hangUpReason;type:longtext;not null" json:"hangUpReason"`
	IsChange                string     `gorm:"column:isChange;type:varchar(25);not null;comment:业务需求是否为变更中" json:"isChange"`
	ChangedReviewers        string     `gorm:"column:changedReviewers;type:mediumtext;not null;comment:变更后审批人" json:"changedReviewers"`
	ChangedBy               string     `gorm:"column:changedBy;type:char(30);default:''" json:"changedBy"`
	IsManagerReview         string     `gorm:"column:isManagerReview;type:varchar(25);default:'wait';comment:是否系统主管审批" json:"isManagerReview"`
	ManagerReviewers        string     `gorm:"column:managerReviewers;type:varchar(255);not null;comment:系统主管审批人" json:"managerReviewers"`
	ManagerReviewResult     *string    `gorm:"column:managerReviewResult;type:text" json:"managerReviewResult"`
	SubmitBy                string     `gorm:"column:submitBy;type:varchar(255);not null;comment:系统主管审批发起人" json:"submitBy"`
	IsReturned              string     `gorm:"column:isReturned;type:enum('0','1');default:'0';comment:是否退回 0-否 1-是" json:"isReturned"`
	ReturnedReason          *string    `gorm:"column:returnedReason;type:mediumtext;comment:业务需求退回原因" json:"returnedReason"`
	Deadline                *time.Time `gorm:"column:deadline;type:date" json:"deadline"`
	CreatedBy               string     `gorm:"column:createdBy;type:char(30);default:''" json:"createdBy"`
	Originator              string     `gorm:"column:originator;type:char(30);default:'';comment:person who raised the demand" json:"originator"`
	CreatedDate             *time.Time `gorm:"column:createdDate;type:datetime" json:"createdDate"`
	ClarifyDate             *time.Time `gorm:"column:clarifyDate;type:datetime" json:"clarifyDate"`
	ClosedBy                string     `gorm:"column:closedBy;type:char(30);default:''" json:"closedBy"`
	ClosedDate              *time.Time `gorm:"column:closedDate;type:datetime" json:"closedDate"`
	ClosedReason            string     `gorm:"column:closedReason;type:varchar(30);default:''" json:"closedReason"`
	Mailto                  *string    `gorm:"column:mailto;type:text" json:"mailto"`
	Deleted                 string     `gorm:"column:deleted;type:enum('0','1');default:'0'" json:"deleted"`
	ClarifyDesc             string     `gorm:"column:clarifyDesc;type:mediumtext;not null" json:"clarifyDesc"`
	Story                   int        `gorm:"column:story;default:0" json:"story"`
	ReviewedBy              *string    `gorm:"column:reviewedBy;type:text" json:"reviewedBy"`
	ReviewedDate            *time.Time `gorm:"column:reviewedDate;type:datetime" json:"reviewedDate"`
	FinalStatus             string     `gorm:"column:finalStatus;type:char(30);not null" json:"finalStatus"`
	DuplicateDemand         *int       `gorm:"column:duplicateDemand;type:mediumint" json:"duplicateDemand"`
	SubmitAcceptanceDate    *time.Time `gorm:"column:submitAcceptanceDate;type:datetime" json:"submitAcceptanceDate"`
	Acceptance              *string    `gorm:"column:acceptance;type:varchar(25)" json:"acceptance"`
	AcceptancedDate         *time.Time `gorm:"column:acceptancedDate;type:datetime" json:"acceptancedDate"`
	Accepter                string     `gorm:"column:accepter;type:char(30);not null;comment:执行验收人" json:"accepter"`
	Comment                 *string    `gorm:"column:comment;type:mediumtext" json:"comment"`
	EstimateLaunch          *time.Time `gorm:"column:estimateLaunch;type:date" json:"estimateLaunch"`
	SchedulePlanDate        *time.Time `gorm:"column:schedulePlanDate;type:date" json:"schedulePlanDate"`
	ActualDevStartDate      *time.Time `gorm:"column:actualDevStartDate;type:date" json:"actualDevStartDate"`
	ActualTestStartDate     *time.Time `gorm:"column:actualTestStartDate;type:date" json:"actualTestStartDate"`
	EstimateLaunchChange    *time.Time `gorm:"column:estimateLaunchChange;type:date" json:"estimateLaunchChange"`
	DeliverDate             *time.Time `gorm:"column:deliverDate;type:date" json:"deliverDate"`
	PublishWindow           *time.Time `gorm:"column:publishWindow;type:date" json:"publishWindow"`
	MainDevelopers          string     `gorm:"column:mainDevelopers;type:varchar(255);default:''" json:"mainDevelopers"`
	MainTesters             string     `gorm:"column:mainTesters;type:varchar(255);default:''" json:"mainTesters"`
	OnetimeAcceptance       string     `gorm:"column:onetimeAcceptance;type:varchar(10);not null" json:"onetimeAcceptance"`
	ReviewMark              string     `gorm:"column:reviewMark;type:varchar(25);default:''" json:"reviewMark"`
	IsNeedReview            string     `gorm:"column:isNeedReview;type:varchar(25);not null" json:"isNeedReview"`
	LeadDept                string     `gorm:"column:leadDept;type:varchar(25);not null" json:"leadDept"`
	DevelopFinish           *time.Time `gorm:"column:developFinish;type:date" json:"developFinish"`
	DevelopFinishChange     *time.Time `gorm:"column:developFinishChange;type:date" json:"developFinishChange"`
	TestFinish              *time.Time `gorm:"column:testFinish;type:date" json:"testFinish"`
	TestFinishChange        *time.Time `gorm:"column:testFinishChange;type:date" json:"testFinishChange"`
	VerifyFinish            *time.Time `gorm:"column:verifyFinish;type:date" json:"verifyFinish"`
	VerifyFinishChange      *time.Time `gorm:"column:verifyFinishChange;type:date" json:"verifyFinishChange"`
	Feedback                int        `gorm:"column:feedback;default:0" json:"feedback"`
	EditedBy                string     `gorm:"column:editedBy;type:char(30);not null" json:"editedBy"`
	EditedDate              *time.Time `gorm:"column:editedDate;type:datetime" json:"editedDate"`
	IsNeedFocus             string     `gorm:"column:isNeedFocus;type:enum('0','1');default:'0';comment:是否需要重点关注" json:"isNeedFocus"`
	ChangeReason            string     `gorm:"column:changeReason;type:longtext;not null;comment:变更原因" json:"changeReason"`
	DispatchTime            *time.Time `gorm:"column:dispatchTime;type:datetime" json:"dispatchTime"`
	IsDispatched            string     `gorm:"column:isDispatched;type:enum('0','1');default:'0';comment:是否已需求分派" json:"isDispatched"`
	DemandCompletionTime    *time.Time `gorm:"column:demandCompletionTime;type:date" json:"demandCompletionTime"`
	Color                   string     `gorm:"column:color;type:varchar(255);default:''" json:"color"`
	OriginalStatus          string     `gorm:"column:originalStatus;type:varchar(25);not null;comment:原状态" json:"originalStatus"`
	ProposeDept             string     `gorm:"column:proposeDept;type:varchar(255);not null" json:"proposeDept"`
	DemandSize              string     `gorm:"column:demandSize;type:varchar(255);default:''" json:"demandSize"`
	Overall                 string     `gorm:"column:overall;type:enum('0','1','2','3','4','5');default:'0'" json:"overall"`
	DeliveryCycle           string     `gorm:"column:deliveryCycle;type:enum('0','1','2','3','4','5');default:'0'" json:"deliveryCycle"`
	DeliveryQuality         string     `gorm:"column:deliveryQuality;type:enum('0','1','2','3','4','5');default:'0'" json:"deliveryQuality"`
	Collabora               string     `gorm:"column:collabora;type:enum('0','1','2','3','4','5');default:'0'" json:"collabora"`
	AppraiseTime            *time.Time `gorm:"column:appraiseTime;type:datetime" json:"appraiseTime"`
	AppraiseDesc            string     `gorm:"column:appraiseDesc;type:mediumtext;not null;comment:评价描述" json:"appraiseDesc"`
	AppraiseBy              string     `gorm:"column:appraiseBy;type:char(30);not null;comment:评价人" json:"appraiseBy"`
	ReasonType              string     `gorm:"column:reasonType;type:varchar(255);default:''" json:"reasonType"`
	DelaySystem             *string    `gorm:"column:delaySystem;type:text" json:"delaySystem"`
	DevPostponed            int        `gorm:"column:devPostponed;type:tinyint(1);default:0;comment:开发延期" json:"devPostponed"`
	TestPostponed           int        `gorm:"column:testPostponed;type:tinyint(1);default:0;comment:测试延期" json:"testPostponed"`
	VerifyPostponed         int        `gorm:"column:verifyPostponed;type:tinyint(1);default:0;comment:验收延期" json:"verifyPostponed"`
	LaunchPostponed         int        `gorm:"column:launchPostponed;type:tinyint(1);default:0;comment:上线延期" json:"launchPostponed"`
	Keywords                string     `gorm:"column:keywords;type:varchar(255);default:''" json:"keywords"`
	Product                 string     `gorm:"column:product;type:varchar(255);default:''" json:"product"`
	Title                   string     `gorm:"column:title;type:varchar(255);default:''" json:"title"`
	FeedbackedBy            string     `gorm:"column:feedbackedBy;type:varchar(255);default:''" json:"feedbackedBy"`
	AssignedDate            *time.Time `gorm:"column:assignedDate;type:datetime" json:"assignedDate"`
	Duration                string     `gorm:"column:duration;type:char(30);default:''" json:"duration"`
	BSA                     string     `gorm:"column:BSA;type:char(30);default:''" json:"bsa"`
	Roadmap                 int        `gorm:"column:roadmap;default:0" json:"roadmap"`
	ChildDemands            string     `gorm:"column:childDemands;type:varchar(255);default:''" json:"childDemands"`
	Version                 string     `gorm:"column:version;type:varchar(255);default:''" json:"version"`
	ParentVersion           int        `gorm:"column:parentVersion;default:0" json:"parentVersion"`
	Vision                  string     `gorm:"column:vision;type:varchar(255);default:'or'" json:"vision"`
	ChangedDate             *time.Time `gorm:"column:changedDate;type:datetime" json:"changedDate"`
	SubmitedBy              string     `gorm:"column:submitedBy;type:varchar(30);default:''" json:"submitedBy"`
	LastEditedBy            string     `gorm:"column:lastEditedBy;type:varchar(30);default:''" json:"lastEditedBy"`
	LastEditedDate          *time.Time `gorm:"column:lastEditedDate;type:datetime" json:"lastEditedDate"`
	ActivatedDate           *time.Time `gorm:"column:activatedDate;type:datetime" json:"activatedDate"`
	DistributedBy           string     `gorm:"column:distributedBy;type:varchar(30);default:''" json:"distributedBy"`
	DistributedDate         *time.Time `gorm:"column:distributedDate;type:datetime" json:"distributedDate"`
	IsCarReview             string     `gorm:"column:isCarReview;type:enum('0','1');default:'0';comment:is car mode" json:"isCarReview"`
	IsGrayVerifyPlan        string     `gorm:"column:isGrayVerifyPlan;type:char(2);default:'';comment:灰度验证计划" json:"isGrayVerifyPlan"`
	IsNewProduct            string     `gorm:"column:isNewProduct;type:char(10);default:'-1';comment:is new product" json:"isNewProduct"`
	IsRelatedAccounts       string     `gorm:"column:isRelatedAccounts;type:char(10);default:'-1';comment:is related accounts" json:"isRelatedAccounts"`
	IsNewFunction           string     `gorm:"column:isNewFunction;type:char(10);default:'-1';comment:is new function" json:"isNewFunction"`
	IsImportantOrder        string     `gorm:"column:isImportantOrder;type:enum('0','1');default:'0';comment:is important Order" json:"isImportantOrder"`
	IsAutoRecImportantOrder string     `gorm:"column:isAutoRecImportantOrder;type:enum('0','1');default:'0';comment:auto recognize important order flag" json:"isAutoRecImportantOrder"`
	IsOtherImportantOrder   string     `gorm:"column:isOtherImportantOrder;type:char(10);default:'-1';comment:is other important Order" json:"isOtherImportantOrder"`
	MultiLegalPersonLogo    string     `gorm:"column:multiLegalPersonLogo;type:char(10);default:'-1';comment:multi legal person logo" json:"multiLegalPersonLogo"`
	RelevantManagers        string     `gorm:"column:relevantManagers;type:varchar(255);not null;comment:notify relevant department managers for acceptance" json:"relevantManagers"`
	TeamGroup               string     `gorm:"column:teamGroup;type:varchar(25);default:'';comment:teamGroup" json:"teamGroup"`
	EstimateDelivery        int        `gorm:"column:estimateDelivery;default:0;comment:demand estimate delivery" json:"estimateDelivery"`
	DfBy                    string     `gorm:"column:dfBy;type:varchar(30);not null" json:"dfBy"`
	DfDate                  *time.Time `gorm:"column:dfDate;type:datetime" json:"dfDate"`
	DddBy                   string     `gorm:"column:dddBy;type:varchar(30);not null" json:"dddBy"`
	DddDate                 *time.Time `gorm:"column:dddDate;type:datetime" json:"dddDate"`
	ScaleEstimation         string     `gorm:"column:scaleEstimation;type:varchar(255);default:''" json:"scaleEstimation"`
	Participants            string     `gorm:"column:partiCipants;type:varchar(255);default:''" json:"participants"`
	VerifyDate              string     `gorm:"column:verifyDate;type:varchar(30);default:''" json:"verifyDate"`
	VerifyPlan              string     `gorm:"column:verifyPlan;type:text;not null" json:"verifyPlan"`
	Verifier                string     `gorm:"column:veriFier;type:varchar(255);default:''" json:"verifier"`
	ArchReviewer            *string    `gorm:"column:archReviewer;type:varchar(255)" json:"archReviewer"`
	HangUpType              string     `gorm:"column:hangUpType;type:char(5);default:''" json:"hangUpType"`
	QualityCheck            string     `gorm:"column:qualityCheck;type:varchar(255);default:''" json:"qualityCheck"`
	CanDeliver              string     `gorm:"column:canDeliver;type:char(10);default:'';comment:是否可交付 yes-可交付 no-不可交付" json:"canDeliver"`
}

// TableName 指定表名
func (ZtDemand) TableName() string {
	return "zt_demand"
}
