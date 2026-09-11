<?php
helper::importControl('demand');
class myDemand extends demand
{
    public function submit($demandID = 0)
    {
        if(!$_POST) return parent::submit($demandID);

        $changes = $this->demand->submitReview($demandID);
        if(dao::isError())
        {
            return $this->send(array('result' => 'fail', 'message' => dao::getError()));
        }
        if($changes)
        {
            $actionID = $this->loadModel('action')->create('demand', $demandID, 'submited', $this->post->comment);
            $this->action->logHistory($actionID, $changes);
            $this->action->syncOA($actionID, array('create' => $this->post->reviewer));
        }

        // In API mode send() returns; stop before the parent's GET form rendering.
        return $this->send(array('result' => 'success', 'message' => $this->lang->saveSuccess, 'load' => true));
    }
}
