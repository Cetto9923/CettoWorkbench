<?php
require_once __DIR__ . "/demandreviewresponse.php";
class demandReviewEntry extends entry
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
        $fields = 'result,mailto,comment,isNeedFocus';
        $this->batchSetPost($fields);

        $control = $this->loadController('demand', 'review');
        // Native API control::send consumes one output buffer before printing its JSON.
        ob_start();
        $control->review($demandID);

        return demandReviewResponse($this, $this->getData());
    }
}
