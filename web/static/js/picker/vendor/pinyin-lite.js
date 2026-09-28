/* =============================================================================
 * 文件: web/static/workbench/shared/picker/vendor/pinyin-lite.js
 * 职责: shared/picker 拼音 vendor（Plan §11）
 *
 * 维护约束（Plan §22 #10）：不自研完整拼音字典；扩展请新增单字到 SINGLE 字典。
 * 许可证: MIT-0（同目录 LICENSE）
 *
 * 说明：本文件保留接口契约 + 文本 fallback + 高频姓氏/常用字精简表。
 * ============================================================================= */
(function (global) {
  'use strict';

  var SINGLE = {
    张:'zhang',王:'wang',李:'li',赵:'zhao',刘:'liu',陈:'chen',杨:'yang',黄:'huang',
    周:'zhou',吴:'wu',徐:'xu',孙:'sun',胡:'hu',朱:'zhu',高:'gao',林:'lin',
    何:'he',郭:'guo',马:'ma',罗:'luo',梁:'liang',宋:'song',郑:'zheng',谢:'xie',
    韩:'han',唐:'tang',冯:'feng',于:'yu',董:'dong',蒋:'jiang',苏:'su',叶:'ye',
    吕:'lv',杜:'du',魏:'wei',程:'cheng',丁:'ding',任:'ren',沈:'shen',姚:'yao',
    卢:'lu',姜:'jiang',崔:'cui',钟:'zhong',谭:'tan',陆:'lu',汪:'wang',范:'fan',
    金:'jin',石:'shi',廖:'liao',贾:'jia',夏:'xia',韦:'wei',付:'fu',方:'fang',
    白:'bai',邹:'zou',孟:'meng',熊:'xiong',秦:'qin',邱:'qiu',尹:'yin',薛:'xue',
    闫:'yan',段:'duan',雷:'lei',侯:'hou',龙:'long',史:'shi',陶:'tao',黎:'li',
    贺:'he',顾:'gu',毛:'mao',郝:'hao',龚:'gong',邵:'shao',钱:'qian',孔:'kong',
    向:'xiang',汤:'tang',洪:'hong',常:'chang',万:'wan',潘:'pan',邓:'deng',
    伟:'wei',芳:'fang',娜:'na',强:'qiang',杰:'jie',磊:'lei',敏:'min',静:'jing',
    丽:'li',军:'jun',健:'jian',超:'chao',涛:'tao',峰:'feng',辉:'hui',勇:'yong',
    国:'guo',江:'jiang',河:'he',湖:'hu',山:'shan',川:'chuan',明:'ming',华:'hua',
    建:'jian',平:'ping',产:'chan',研:'yan',发:'fa',银:'yin',行:'xing',数:'shu',
    经:'jing',理:'li',业:'ye',总:'zong',监:'jian',师:'shi',学:'xue',长:'zhang',
    部:'bu',门:'men',团:'tuan',校:'xiao',生:'sheng'
  };

  function isAsciiLetter(ch) {
    return /^[a-zA-Z]$/.test(ch);
  }

  function lookupChar(ch) {
    if (isAsciiLetter(ch)) return ch.toLowerCase();
    return SINGLE[ch] || null;
  }

  function forText(str) {
    var out = [];
    if (!str) return out;
    var s = String(str);
    var buf = '';
    function flushAscii() {
      if (buf) { out.push({ c: buf, py: buf.toLowerCase() }); buf = ''; }
    }
    for (var i = 0; i < s.length; i++) {
      var ch = s.charAt(i);
      if (/[a-zA-Z0-9]/.test(ch)) {
        buf += ch.toLowerCase();
        continue;
      }
      flushAscii();
      var py = lookupChar(ch);
      out.push({ c: ch, py: py != null ? py : ch });
    }
    flushAscii();
    return out;
  }

  function fullPinyin(str) {
    return forText(str).map(function (x) { return x.py; }).join(' ');
  }
  function initials(str) {
    return forText(str).map(function (x) {
      return /^[a-zA-Z]+$/.test(x.py) ? x.py.charAt(0) : '';
    }).join('');
  }

  global.PinyinLite = {
    forText: forText,
    fullPinyin: fullPinyin,
    initials: initials,
    _SIZE: Object.keys(SINGLE).length
  };
})(typeof window !== 'undefined' ? window : globalThis);
