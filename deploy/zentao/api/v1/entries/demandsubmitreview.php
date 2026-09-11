<?php
require_once __DIR__ . "/demandreviewresponse.php";
class demandSubmitReviewEntry extends entry
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
        $fields = 'reviewer,comment';
        $this->batchSetPost($fields);

        $control = $this->loadController('demand', 'submit');
        // Native API control::send consumes one output buffer before printing its JSON.
        ob_start();
        $control->submit($demandID);

        return demandReviewResponse($this, $this->getData());
    }
}
