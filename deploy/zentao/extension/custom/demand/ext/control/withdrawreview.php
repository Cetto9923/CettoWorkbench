<?php
helper::importControl('demand');
class myDemand extends demand
{
    public function withdrawReview($demandID = 0)
    {
        if($_POST)
        {
            $this->demand->withdrawReview($demandID);
            if(dao::isError())
            {
                return $this->send(array('result' => 'fail', 'message' => dao::getError()));
            }

            // In API mode send() returns; stop before the GET form rendering.
            return $this->send(array('result' => 'success', 'message' => $this->lang->saveSuccess, 'load' => true));
        }

        $this->view->title   = $this->lang->demand->withdrawReview;
        $this->view->demand  = $this->demand->getByID($demandID);
        $this->view->actions = $this->loadModel('action')->getList('demand', $demandID);
        $this->display();
    }
}
