<?php
class demandAIGenerateEntry extends entry
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
        $fields = 'inputContent,source';
        $this->batchSetPost($fields);

        $inputContent = isset($this->post->inputContent) ? $this->post->inputContent : '';
        $source       = isset($this->post->source) && $this->post->source ? $this->post->source : 'clarify';
        $demandID     = intval($demandID);

        $aiModel  = $this->loadModel('ai');
        $AIPrompt = $aiModel->getPublishedAIByMethod('demand.' . $source);
        if(empty($AIPrompt))
        {
            $AIPrompt = $aiModel->getPublishedAIByMethod('demand.create');
        }

        $content      = '';
        $thinkContent = '';

        if(!empty($AIPrompt))
        {
            try
            {
                $response      = $aiModel->chatByPrompt($AIPrompt, $inputContent);
                $outputContent = isset($response[0]) ? $response[0] : '';
                $content       = $outputContent;

                if(preg_match('/<think>(.*?)<\/think>/s', $outputContent, $matches))
                {
                    $thinkContent = trim($matches[1]);
                    $content      = trim(str_replace($matches[0], '', $outputContent));
                }
            }
            catch(Throwable $e)
            {
                $content = '';
            }
        }

        if(empty($content))
        {
            return $this->sendError(400, "禅道未配置大模型或AI服务未启用，无法自动生成用户故事");
        }

        $demandModel = $this->loadModel('demand');
        $demandModel->statisticsAI($source);

        $aiCodes = array();
        if($source == 'clarify' && $demandID > 0)
        {
            $aiCodes = $demandModel->saveAIUserStory($demandID, $content);
        }

        return $this->send(200, array(
            'result'       => 'success',
            'content'      => $content,
            'thinkContent' => $thinkContent,
            'aiCodes'      => $aiCodes
        ));
    }
}
