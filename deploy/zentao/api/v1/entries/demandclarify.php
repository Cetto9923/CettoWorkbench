<?php
class demandClarifyEntry extends entry
{
    /**
     * POST method.
     *
     * @param  int $demandID
     * @access public
     * @return string
     */
    public function post($demandID)
    {
        $fields = 'category,BRA,QD,RD,products,PM,demandCompletionDate,systemClarifyDesc,isAdditionalInfo,additionalInfo,isMainSystem,scaleEstimation,userStoryChecked,userStoryID,userStoryNO,role,gv,entryProductID,point,sourceType,revpoint,aiCode,clarifyDesc,isNewProduct,isRelatedAccounts,isNewFunction,isOtherImportantOrder,multiLegalPersonLogo,status,comment,id';
        $this->batchSetPost($fields);

        // 规范化复杂数组结构，确保与禅道原生 $_POST 格式完全一致
        if(isset($_POST['isMainSystem']))
        {
            $_POST['isMainSystem'] = (array)$_POST['isMainSystem'];
        }
        if(isset($_POST['userStoryChecked']))
        {
            $_POST['userStoryChecked'] = (array)$_POST['userStoryChecked'];
        }
        if(isset($_POST['isAdditionalInfo']))
        {
            $info = (array)$_POST['isAdditionalInfo'];
            foreach($info as $k => $v)
            {
                if(!is_array($v)) $info[$k] = array($v);
            }
            $_POST['isAdditionalInfo'] = $info;
        }
        if(isset($_POST['multiLegalPersonLogo']))
        {
            $logo = strval($_POST['multiLegalPersonLogo']);
            if($logo === 'changshu') $logo = '0';
            elseif($logo === 'village') $logo = '1';
            elseif($logo === 'changshu_village') $logo = '2';
            $_POST['multiLegalPersonLogo'] = $logo;
        }
        if(isset($_POST['id']))
        {
            $idList = array();
            foreach((array)$_POST['id'] as $k => $v)
            {
                if(!empty($v) && intval($v) > 0)
                {
                    $idList[$k] = intval($v);
                }
            }
            $_POST['id'] = $idList;
        }

        $demandModel = $this->loadModel('demand');
        $changes     = $demandModel->clarify($demandID);

        if(dao::isError())
        {
            $errors = dao::getError();
            $messages = array();
            if(is_array($errors))
            {
                foreach($errors as $field => $err)
                {
                    if(is_array($err))
                    {
                        $messages[] = (is_string($field) ? $field . ': ' : '') . implode(', ', $err);
                    }
                    else
                    {
                        $messages[] = (is_string($field) ? $field . ': ' : '') . (string)$err;
                    }
                }
            }
            else
            {
                $messages[] = (string)$errors;
            }
            $msg = implode('; ', $messages);
            return $this->sendError(400, $msg);
        }

        $actionModel = $this->loadModel('action');
        $comment     = isset($this->post->comment) ? $this->post->comment : '';
        $actionID    = $actionModel->create('demand', $demandID, 'clarify', $comment);
        if($changes)
        {
            $actionModel->logHistory($actionID, $changes);
        }

        $msg = $this->lang->saveSuccess;
        if(isset($this->post->isNewProduct) && $this->post->isNewProduct == '1')
        {
            $msg = $this->lang->demand->newProductInfo;
        }

        return $this->send(200, array('message' => $msg));
    }
}
