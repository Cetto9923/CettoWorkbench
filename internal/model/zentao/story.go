// =============================================================================
// 文件: internal/model/zentao/story.go
// 模块: 数据模型
// 类型: model
// 职责: 定义需求模型字段与表映射。
// 依赖: 无
// =============================================================================

package model

import (
    "time"
)

// ZtStory 需求/故事表模型
type ZtStory struct {
    ID                       uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Vision                   string     `gorm:"column:vision;type:varchar(10);default:'rnd'" json:"vision"`
    Parent                   int        `gorm:"column:parent;default:0" json:"parent"`
    IsParent                 string     `gorm:"column:isParent;type:enum('0','1');default:'0'" json:"isParent"`
    Root                     int        `gorm:"column:root;default:0" json:"root"`
    Path                     *string    `gorm:"column:path;type:text" json:"path"`
    Grade                    int        `gorm:"column:grade;default:0" json:"grade"`
    Product                  uint       `gorm:"column:product;default:0" json:"product"`
    Branch                   uint       `gorm:"column:branch;default:0" json:"branch"`
    Module                   uint       `gorm:"column:module;default:0" json:"module"`
    FromDemand               uint       `gorm:"column:fromDemand;default:0" json:"fromDemand"`
    Plan                     *string    `gorm:"column:plan;type:mediumtext" json:"plan"`
    Roadmap                  string     `gorm:"column:roadmap;type:varchar(255);default:''" json:"roadmap"`
    Source                   string     `gorm:"column:source;type:varchar(20);default:''" json:"source"`
    SourceNote               string     `gorm:"column:sourceNote;type:varchar(255);default:''" json:"sourceNote"`
    FromBug                  uint       `gorm:"column:fromBug;default:0" json:"fromBug"`
    Feedback                 uint       `gorm:"column:feedback;default:0" json:"feedback"`
    Title                    string     `gorm:"column:title;type:varchar(255);default:''" json:"title"`
    Keywords                 string     `gorm:"column:keywords;type:varchar(255);default:''" json:"keywords"`
    Type                     string     `gorm:"column:type;type:varchar(30);default:'story'" json:"type"`
    Category                 string     `gorm:"column:category;type:varchar(30);default:'feature'" json:"category"`
    Pri                      uint8      `gorm:"column:pri;type:tinyint unsigned;default:3" json:"pri"`
    Estimate                 float32    `gorm:"column:estimate;type:float unsigned;default:0" json:"estimate"`
    Status                   string     `gorm:"column:status;type:enum('','changing','active','draft','closed','reviewing','launched','developing');default:''" json:"status"`
    SubStatus                string     `gorm:"column:subStatus;type:varchar(30);default:''" json:"subStatus"`
    Color                    string     `gorm:"column:color;type:char(7);default:''" json:"color"`
    Stage                    string     `gorm:"column:stage;type:enum('','wait','inroadmap','incharter','planned','projected','designing','designed','developing','developed','testing','tested','verified','rejected','delivering','delivered','released','closed');default:'wait'" json:"stage"`
    StagedBy                 string     `gorm:"column:stagedBy;type:char(30);default:''" json:"stagedBy"`
    Mailto                   *string    `gorm:"column:mailto;type:mediumtext" json:"mailto"`
    Lib                      uint       `gorm:"column:lib;default:0" json:"lib"`
    FromStory                uint       `gorm:"column:fromStory;default:0" json:"fromStory"`
    FromVersion              int        `gorm:"column:fromVersion;default:1" json:"fromVersion"`
    OpenedBy                 string     `gorm:"column:openedBy;type:varchar(30);default:''" json:"openedBy"`
    OpenedDate               *time.Time `gorm:"column:openedDate;type:datetime" json:"openedDate"`
    AssignedTo               string     `gorm:"column:assignedTo;type:varchar(30);default:''" json:"assignedTo"`
    AssignedDate             *time.Time `gorm:"column:assignedDate;type:datetime" json:"assignedDate"`
    ApprovedDate             *time.Time `gorm:"column:approvedDate;type:date" json:"approvedDate"`
    LastEditedBy             string     `gorm:"column:lastEditedBy;type:varchar(30);default:''" json:"lastEditedBy"`
    LastEditedDate           *time.Time `gorm:"column:lastEditedDate;type:datetime" json:"lastEditedDate"`
    ChangedBy                string     `gorm:"column:changedBy;type:varchar(30);default:''" json:"changedBy"`
    ChangedDate              *time.Time `gorm:"column:changedDate;type:datetime" json:"changedDate"`
    SubmitedBy               *string    `gorm:"column:submitedBy;type:varchar(30)" json:"submitedBy"`
    ReviewedBy               string     `gorm:"column:reviewedBy;type:varchar(255);default:''" json:"reviewedBy"`
    ReviewedDate             *time.Time `gorm:"column:reviewedDate;type:datetime" json:"reviewedDate"`
    ReleasedDate             *time.Time `gorm:"column:releasedDate;type:datetime" json:"releasedDate"`
    ClosedBy                 string     `gorm:"column:closedBy;type:varchar(30);default:''" json:"closedBy"`
    ClosedDate               *time.Time `gorm:"column:closedDate;type:datetime" json:"closedDate"`
    ClosedReason             string     `gorm:"column:closedReason;type:varchar(30);default:''" json:"closedReason"`
    ActivatedDate            *time.Time `gorm:"column:activatedDate;type:datetime" json:"activatedDate"`
    ToBug                    uint       `gorm:"column:toBug;default:0" json:"toBug"`
    LinkStories              string     `gorm:"column:linkStories;type:varchar(255);default:''" json:"linkStories"`
    LinkRequirements         string     `gorm:"column:linkRequirements;type:varchar(255);default:''" json:"linkRequirements"`
    Twins                    string     `gorm:"column:twins;type:varchar(255);default:''" json:"twins"`
    DuplicateStory           uint       `gorm:"column:duplicateStory;default:0" json:"duplicateStory"`
    Version                  int        `gorm:"column:version;default:1" json:"version"`
    ParentVersion            int        `gorm:"column:parentVersion;default:0" json:"parentVersion"`
    DemandVersion            int        `gorm:"column:demandVersion;default:0" json:"demandVersion"`
    StoryChanged             string     `gorm:"column:storyChanged;type:enum('0','1');default:'0'" json:"storyChanged"`
    FeedbackBy               string     `gorm:"column:feedbackBy;type:varchar(100);default:''" json:"feedbackBy"`
    NotifyEmail              string     `gorm:"column:notifyEmail;type:varchar(100);default:''" json:"notifyEmail"`
    Demand                   *int       `gorm:"column:demand;type:mediumint;default:0" json:"demand"`
    Duration                 *string    `gorm:"column:duration;type:char(30)" json:"duration"`
    BSA                      *string    `gorm:"column:BSA;type:char(30)" json:"bsa"`
    URChanged                string     `gorm:"column:URChanged;type:enum('0','1');default:'0'" json:"urChanged"`
    Deleted                  string     `gorm:"column:deleted;type:enum('0','1');default:'0'" json:"deleted"`
    SourceType               string     `gorm:"column:sourceType;type:varchar(255);not null" json:"sourceType"`
    EstimateLaunch           *time.Time `gorm:"column:estimateLaunch;type:date" json:"estimateLaunch"`
    EstimateDevCompletion    *time.Time `gorm:"column:estimateDevCompletion;type:date" json:"estimateDevCompletion"`
    DeliverDate              *time.Time `gorm:"column:deliverDate;type:date" json:"deliverDate"`
    ActualDevCompletion      *time.Time `gorm:"column:actualDevCompletion;type:date" json:"actualDevCompletion"`
    DevelopFinish            *time.Time `gorm:"column:developFinish;type:date" json:"developFinish"`
    TestFinish               *time.Time `gorm:"column:testFinish;type:date" json:"testFinish"`
    VerifyFinish             *time.Time `gorm:"column:verifyFinish;type:date" json:"verifyFinish"`
    AssociatedPublicationID  *int       `gorm:"column:associatedPublicationID;type:mediumint;comment:关联发布ID;值为0说明没有最新关联发布" json:"associatedPublicationID"`
    ReleaseIsDeleted         string     `gorm:"column:releaseIsDeleted;type:enum('','0','1');default:'';comment:发布是否删除" json:"releaseIsDeleted"`
    IsMainSystemAssociation  string     `gorm:"column:isMainSystemAssociation;type:enum('0','1','');default:'';comment:是否为主系统关联" json:"isMainSystemAssociation"`
    RetractedReason          string     `gorm:"column:retractedReason;type:enum('','omit','other');default:''" json:"retractedReason"`
    RetractedBy              string     `gorm:"column:retractedBy;type:varchar(30);default:''" json:"retractedBy"`
    RetractedDate            *time.Time `gorm:"column:retractedDate;type:datetime" json:"retractedDate"`
    VerifiedDate             *time.Time `gorm:"column:verifiedDate;type:datetime" json:"verifiedDate"`
    UnlinkReason             string     `gorm:"column:unlinkReason;type:enum('','omit','other');default:''" json:"unlinkReason"`
    IsCarReview              string     `gorm:"column:isCarReview;type:enum('0','1');default:'0';comment:is car mode" json:"isCarReview"`
    CodeNoChange             string     `gorm:"column:codeNoChange;type:enum('0','1');default:'0'" json:"codeNoChange"`
    VerifyDate               string     `gorm:"column:verifyDate;type:varchar(30);default:''" json:"verifyDate"`
    VerifyPlan               string     `gorm:"column:verifyPlan;type:text;not null" json:"verifyPlan"`
    Verifier                 string     `gorm:"column:veriFier;type:varchar(255);default:''" json:"verifier"`
    HasInterface             string     `gorm:"column:hasInterface;type:varchar(255);default:''" json:"hasInterface"`
}

// TableName 指定表名
func (ZtStory) TableName() string {
    return "zt_story"
}