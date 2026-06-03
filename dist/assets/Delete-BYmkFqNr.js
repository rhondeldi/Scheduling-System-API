import{r as b,j as i}from"./index-CCADbRTz.js";import{g as k,h as w,c as x,d as S,e as M,s as R,m as $,v as m,w as f,k as A}from"./departments-DtsRgcmn.js";import{c as p}from"./TextField-CidhrznA.js";function U(t){return String(t).match(/[\d.\-+]*\s*(.*)/)[1]||""}function j(t){return parseFloat(t)}function z(t){return k("MuiSkeleton",t)}w("MuiSkeleton",["root","text","rectangular","rounded","circular","pulse","wave","withChildren","fitContent","heightAuto"]);const E=t=>{const{classes:e,variant:a,animation:n,hasChildren:s,width:r,height:o}=t;return M({root:["root",a,n,s&&"withChildren",s&&!r&&"fitContent",s&&!o&&"heightAuto"]},z,e)},l=m`
  0% {
    opacity: 1;
  }

  50% {
    opacity: 0.4;
  }

  100% {
    opacity: 1;
  }
`,h=m`
  0% {
    transform: translateX(-100%);
  }

  50% {
    /* +0.5s of delay between each loop */
    transform: translateX(100%);
  }

  100% {
    transform: translateX(100%);
  }
`,I=typeof l!="string"?f`
        animation: ${l} 2s ease-in-out 0.5s infinite;
      `:null,D=typeof h!="string"?f`
        &::after {
          animation: ${h} 2s linear 0.5s infinite;
        }
      `:null,V=R("span",{name:"MuiSkeleton",slot:"Root",overridesResolver:(t,e)=>{const{ownerState:a}=t;return[e.root,e[a.variant],a.animation!==!1&&e[a.animation],a.hasChildren&&e.withChildren,a.hasChildren&&!a.width&&e.fitContent,a.hasChildren&&!a.height&&e.heightAuto]}})($(({theme:t})=>{const e=U(t.shape.borderRadius)||"px",a=j(t.shape.borderRadius);return{display:"block",backgroundColor:t.vars?t.vars.palette.Skeleton.bg:A(t.palette.text.primary,t.palette.mode==="light"?.11:.13),height:"1.2em",variants:[{props:{variant:"text"},style:{marginTop:0,marginBottom:0,height:"auto",transformOrigin:"0 55%",transform:"scale(1, 0.60)",borderRadius:`${a}${e}/${Math.round(a/.6*10)/10}${e}`,"&:empty:before":{content:'"\\00a0"'}}},{props:{variant:"circular"},style:{borderRadius:"50%"}},{props:{variant:"rounded"},style:{borderRadius:(t.vars||t).shape.borderRadius}},{props:({ownerState:n})=>n.hasChildren,style:{"& > *":{visibility:"hidden"}}},{props:({ownerState:n})=>n.hasChildren&&!n.width,style:{maxWidth:"fit-content"}},{props:({ownerState:n})=>n.hasChildren&&!n.height,style:{height:"auto"}},{props:{animation:"pulse"},style:I||{animation:`${l} 2s ease-in-out 0.5s infinite`}},{props:{animation:"wave"},style:{position:"relative",overflow:"hidden",WebkitMaskImage:"-webkit-radial-gradient(white, black)","&::after":{background:`linear-gradient(
                90deg,
                transparent,
                ${(t.vars||t).palette.action.hover},
                transparent
              )`,content:'""',position:"absolute",transform:"translateX(-100%)",bottom:0,left:0,right:0,top:0}}},{props:{animation:"wave"},style:D||{"&::after":{animation:`${h} 2s linear 0.5s infinite`}}}]}})),K=b.forwardRef(function(e,a){const n=x({props:e,name:"MuiSkeleton"}),{animation:s="pulse",className:r,component:o="span",height:d,style:g,variant:v="text",width:y,...c}=n,u={...n,animation:s,component:o,variant:v,hasChildren:!!c.children},C=E(u);return i.jsx(V,{as:o,ref:a,className:S(C.root,r),ownerState:u,...c,style:{width:y,height:d,...g}})}),N=p(i.jsx("path",{d:"M3 17.25V21h3.75L17.81 9.94l-3.75-3.75zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34a.996.996 0 0 0-1.41 0l-1.83 1.83 3.75 3.75z"}),"Edit"),T=p(i.jsx("path",{d:"M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6z"}),"Add"),W=p(i.jsx("path",{d:"M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6zM19 4h-3.5l-1-1h-5l-1 1H5v2h14z"}),"Delete");export{T as A,W as D,N as E,K as S};
